# Compliance Posture

- **Cryptography**: The SDK implements no custom cryptographic primitives. It relies on the Go standard library only: TLS through `net/http` for every API call, and HMAC-SHA256 (`crypto/hmac`, `crypto/sha256`, constant-time comparison) for local webhook signature verification.
- **Regulatory**: AML/CFT, sanctions screening and transaction monitoring are performed by QBitFlow's services; the SDK is a client of those services and does not implement them.
- **Data**: The SDK persists nothing and caches nothing. Customer data passes through only in the requests and responses you make, and client-side validation errors never echo the values they reject (for example an email address).
- **Logging**: The SDK does not log. Nothing is written to stdout, stderr or files; what your application logs from SDK results and errors is under your control.
- **Conformance**: The offline test suite (`go test ./...`) pins the SDK's behaviour, including the cross-SDK golden vectors for webhook signatures shared with the JavaScript, Python and PHP SDKs.
