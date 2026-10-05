<a href="https://skillgild.dev/?utm_source=pypi&utm_medium=referral&utm_campaign=sdk"><img src="https://raw.githubusercontent.com/SkillGild/sdks/main/.github/assets/logo.png" alt="SkillGild" width="64" height="64" align="right"></a>

# skillgild

Official Python client for **[SkillGild](https://skillgild.dev/?utm_source=pypi&utm_medium=referral&utm_campaign=sdk)**, the marketplace for hosted AI agent skills. Search the catalog, run hosted skills and drive hybrid tool sessions from Python 3.9+. Standard library only, fully typed (`py.typed`, `TypedDict` responses).

```sh
pip install skillgild
```

Until the first PyPI release: `pip install "git+https://github.com/SkillGild/sdks.git#subdirectory=python"`.

## Usage

```python
import os, uuid
from skillgild import SkillGildClient, SkillGildError

client = SkillGildClient(api_key=os.environ["SKILLGILD_API_KEY"])

# Browse the catalog (no key needed)
page = client.list_skills("video", limit=20)
skill = client.get_skill("logo-design")  # includes media, tools and page copy

# Hybrid skills: your agent follows the guide and calls SkillGild server tools
session = client.start_session("logo-design")
check = client.call_tool(session["session_id"], "validate_svg", {"svg": "<svg .../>"})
client.end_session(session["session_id"])

# Hosted skills run in one call; reuse the idempotency key only when retrying
try:
    run = client.run_skill("some-hosted-skill", {"prompt": "..."}, idempotency_key=str(uuid.uuid4()))
    print(run["output"], run["usage"])
except SkillGildError as error:
    if error.status == 429:
        print("retry in", error.retry_after, "s")
    else:
        raise
```

## API

| Method | Description |
| --- | --- |
| `list_skills(query=None, *, limit=100, cursor=None)` | One catalog page with `next_cursor` |
| `search_skills(query=None)` | First 100 matching skills |
| `get_skill(ref)` | Skill detail, tools, media and page copy |
| `list_categories()` · `list_tags()` · `list_collections()` · `get_collection(slug)` · `featured_skills(slot="home")` | Marketplace content |
| `run_skill(ref, input, *, idempotency_key=None)` | Run a hosted skill |
| `start_session(ref)` · `call_tool(session_id, tool, input)` · `end_session(session_id)` | Hybrid tool sessions |
| `me()` · `revoke_current_key()` | The API key's account; revoke the key |
| `start_device_authorization(name)` · `poll_device_authorization(code)` | Device login that issues an API key |

`SkillGildClient(api_key=None, base_url="https://api.skillgild.dev/v1", timeout=100.0)`. Errors are `SkillGildError` with `status`, `code` and `retry_after`.

Full reference: [SkillGild API documentation](https://skillgild.dev/docs/api?utm_source=pypi&utm_medium=referral&utm_campaign=sdk) · [Skill catalog](https://skillgild.dev/skills?utm_source=pypi&utm_medium=referral&utm_campaign=sdk) · [Source and other languages](https://github.com/SkillGild/sdks)

MIT © SkillGild, Inc.
