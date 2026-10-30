# Operation examples

Each directory contains one runnable operation. Run `go run ./examples/get-info -h` to see the flags, then pass `-run` to allow API requests. Set `EASYSLIP_API_KEY_FILE` to a private key file. `EASYSLIP_V2_BASE_URL` and `EASYSLIP_V1_BASE_URL` optionally override the public API URLs.

Use `-input request.json` to supply arguments. The file contains operation fields such as `payload`, `file`, `id` and `callbackUrl`, never an API key. Examples print response data locally, which can contain personal data. Account and verification examples can change state or consume quota.
