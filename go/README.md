# SkillGild Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/skillgild/sdks/go/skillgild.svg)](https://pkg.go.dev/github.com/skillgild/sdks/go/skillgild)

Official Go client for **[SkillGild](https://skillgild.dev/?utm_source=github&utm_medium=referral&utm_campaign=sdk)**, the marketplace for hosted AI agent skills. Search the catalog, run hosted skills and drive hybrid tool sessions. Go 1.22+, standard library only.

```sh
go get github.com/skillgild/sdks/go
```

```go
import "github.com/skillgild/sdks/go/skillgild"

client, err := skillgild.NewClient("", skillgild.WithAPIKey(os.Getenv("SKILLGILD_API_KEY")))
if err != nil { return err }

// Hybrid skills: your agent follows the guide and calls SkillGild server tools
session, err := client.StartSession(ctx, "logo-design")
if err != nil { return err }
defer client.EndSession(ctx, session.SessionID)
check, err := client.CallTool(ctx, session.SessionID, "validate_svg", map[string]any{"svg": svg})

// Hosted skills run in one call
run, err := client.RunSkill(ctx, "some-hosted-skill", map[string]any{"prompt": "..."},
	skillgild.WithIdempotencyKey(uuid))
var apiErr *skillgild.APIError
if errors.As(err, &apiErr) && apiErr.Status == 429 {
	time.Sleep(apiErr.RetryAfter)
}
```

Methods: `ListSkills`, `SearchSkills`, `GetSkill`, `ListCategories`, `ListTags`, `ListCollections`, `GetCollection`, `FeaturedSkills`, `RunSkill`, `StartSession`, `CallTool`, `EndSession`, `Me`, `RevokeCurrentKey`, `StartDeviceAuthorization`, `PollDeviceAuthorization`. Options: `WithAPIKey`, `WithHTTPClient`.

Full reference: [SkillGild API documentation](https://skillgild.dev/docs/api?utm_source=github&utm_medium=referral&utm_campaign=sdk) · [pkg.go.dev](https://pkg.go.dev/github.com/skillgild/sdks/go/skillgild)

MIT © SkillGild, Inc.
