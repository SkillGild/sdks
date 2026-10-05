import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, HTTPServer

from skillgild import SkillGildClient, SkillGildError


class FakeAPI(BaseHTTPRequestHandler):
    """Records each request and answers with the next queued (status, body, headers)."""

    requests = []
    responses = []

    def _handle(self):
        length = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(length) if length else b""
        FakeAPI.requests.append((self.command, self.path, dict(self.headers), body))
        status, payload, headers = FakeAPI.responses.pop(0) if FakeAPI.responses else (200, {"data": []}, {})
        self.send_response(status)
        for key, value in headers.items():
            self.send_header(key, value)
        raw = payload if isinstance(payload, bytes) else (b"" if payload is None else json.dumps(payload).encode())
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    do_GET = do_POST = do_DELETE = _handle

    def log_message(self, *args):
        pass


class ClientTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = HTTPServer(("127.0.0.1", 0), FakeAPI)
        threading.Thread(target=cls.server.serve_forever, daemon=True).start()
        cls.base = "http://127.0.0.1:%d/v1" % cls.server.server_port

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()

    def setUp(self):
        FakeAPI.requests.clear()
        FakeAPI.responses.clear()

    def client(self, key="sk"):
        return SkillGildClient(api_key=key, base_url=self.base + "/")

    def test_list_skills_unwraps_envelope_without_key(self):
        FakeAPI.responses.append((200, {"data": {"items": [{"slug": "a"}], "next_cursor": "c2"}}, {}))
        page = self.client().list_skills(" design ", limit=10, cursor="c1")
        self.assertEqual(page["next_cursor"], "c2")
        method, path, headers, _ = FakeAPI.requests[0]
        self.assertEqual((method, path), ("GET", "/v1/skills?limit=10&q=design&cursor=c1"))
        self.assertNotIn("Authorization", headers)

    def test_run_sends_key_and_idempotency_key(self):
        FakeAPI.responses.append((200, {"data": {"output": "ok"}}, {}))
        run = self.client().run_skill("a b", {"prompt": "x"}, idempotency_key="k1")
        self.assertEqual(run["output"], "ok")
        method, path, headers, body = FakeAPI.requests[0]
        self.assertEqual((method, path), ("POST", "/v1/skills/a%20b/run"))
        self.assertEqual(headers["Authorization"], "Bearer sk")
        self.assertEqual(headers["Idempotency-Key"], "k1")
        self.assertEqual(json.loads(body), {"input": {"prompt": "x"}})

    def test_key_required_before_request(self):
        with self.assertRaises(ValueError):
            self.client(key=None).me()
        self.assertEqual(FakeAPI.requests, [])

    def test_quota_error_and_retry_after(self):
        FakeAPI.responses.append((429, {"error": {"code": "quota_exceeded", "message": "Monthly runs used"}}, {"Retry-After": "120"}))
        with self.assertRaises(SkillGildError) as caught:
            self.client().run_skill("a", {})
        error = caught.exception
        self.assertEqual((error.status, error.code, str(error), error.retry_after), (429, "quota_exceeded", "Monthly runs used", 120))

    def test_unauthorized_and_non_json_errors(self):
        FakeAPI.responses.append((401, {"error": {"code": "unauthorized", "message": "Invalid API key"}}, {}))
        with self.assertRaises(SkillGildError) as caught:
            self.client().me()
        self.assertEqual((caught.exception.status, caught.exception.code), (401, "unauthorized"))
        FakeAPI.responses.append((502, b"<html>bad gateway</html>", {}))
        with self.assertRaises(SkillGildError) as caught:
            self.client().get_skill("a")
        self.assertEqual((caught.exception.status, caught.exception.code), (502, None))

    def test_no_content(self):
        FakeAPI.responses.append((204, None, {}))
        self.assertIsNone(self.client().end_session("s1"))
        self.assertEqual(FakeAPI.requests[0][:2], ("DELETE", "/v1/skill-sessions/s1"))

    def test_catalog_device_and_key_endpoints(self):
        c = self.client()
        c.list_categories()
        c.list_tags()
        c.list_collections()
        c.get_collection("starter kit")
        c.featured_skills()
        c.start_device_authorization("laptop")
        c.poll_device_authorization("dc")
        c.revoke_current_key()
        self.assertEqual(
            [(m, p) for m, p, _, _ in FakeAPI.requests],
            [
                ("GET", "/v1/categories"),
                ("GET", "/v1/tags"),
                ("GET", "/v1/collections"),
                ("GET", "/v1/collections/starter%20kit"),
                ("GET", "/v1/featured-skills?slot=home"),
                ("POST", "/v1/device-authorizations"),
                ("POST", "/v1/device-authorizations/token"),
                ("DELETE", "/v1/me/api-keys/current"),
            ],
        )
        self.assertEqual(json.loads(FakeAPI.requests[5][3]), {"device_name": "laptop", "client_type": "skillgild-sdk"})

    def test_rejects_bad_base_urls(self):
        for url in ("https://u:p@api.test", "ftp://api.test", "api.test"):
            with self.assertRaises(ValueError):
                SkillGildClient(base_url=url)

    def test_unreachable_api(self):
        with self.assertRaises(SkillGildError) as caught:
            SkillGildClient(base_url="http://127.0.0.1:1/v1", timeout=2).get_skill("a")
        self.assertEqual(caught.exception.status, 0)


if __name__ == "__main__":
    unittest.main()
