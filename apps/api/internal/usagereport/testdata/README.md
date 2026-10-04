# Umami `/api/send` response fixtures (infra-optin-usage-report-a2)

| File                         | Origin                                                                                                                                                                                                               |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `website_not_found_400.json` | **Recorded** 2026-10-04 from the maintainer's self-hosted Umami (`POST /api/send` with an unknown website id). Umami checks the website before the bot filter, so this is also what a mis-configured release gets.   |
| `bot_dropped_200.json`       | Shape from Umami source (`src/app/api/send/route.ts`, master = v3.4.0): a request its `isbot` check flags is answered **HTTP 200** with this body and nothing is recorded. Not recordable without a real website id. |
| `accepted_200.json`          | Shape from the same source file (`{cache, sessionId, visitId}`); ids are placeholders. Not recordable without writing a real event.                                                                                  |

Replace the two source-derived fixtures with recordings once the Vido website exists in Umami (`ops-umami-usage-report-prep`).
