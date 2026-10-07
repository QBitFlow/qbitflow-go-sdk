package qbf

// Version is the SDK release version. It is sent on every request as the User-Agent
// (`qbitflow-go/<Version>`) and must be bumped together with CHANGELOG.md and the git tag.
const Version = "2.5.0"

// userAgent is the User-Agent header value sent on every request.
const userAgent = "qbitflow-go/" + Version
