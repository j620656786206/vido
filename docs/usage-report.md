# Anonymous usage report (opt-in)

[繁體中文](usage-report.zh-TW.md)

Vido can send one small anonymous count per week to its maintainer, so he can see whether Vido is actually being relied on. **It is off by default** and stays off unless you turn it on.

## What is sent

Once every 7 days, at most, while the report is on:

| Field                   | Example                                | What it is                                                                                                                                                                                    |
| ----------------------- | -------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                    | `3f2a6c1e-8f0b-4d5e-9a7c-1b2c3d4e5f60` | A random ID created the first time you turn the report on. It is not derived from your hardware, account, network or anything else, and it is the same every week so one machine counts once. |
| `version`               | `0.1.2`                                | The Vido version you run.                                                                                                                                                                     |
| `subtitles_auto_7d`     | `7`                                    | How many subtitles Vido produced **on its own** in the last 7 days (after a scan, or after a download finished). Subtitles you started yourself are not counted.                              |
| `subtitles_embedded_7d` | `4`                                    | …of which came from a track already inside the video.                                                                                                                                         |
| `subtitles_online_7d`   | `2`                                    | …of which came from an online subtitle source.                                                                                                                                                |
| `subtitles_asr_7d`      | `1`                                    | …of which came from speech recognition.                                                                                                                                                       |

The other fields are fixed values the receiver needs and are the same for every Vido: `website`, `hostname` (`vido`), `url` (`/usage-report`), `name` (`weekly_usage`), and `ip` set to `127.0.0.1`, which asks the receiver not to look up your country (current Umami versions skip the lookup for it; the maintainer checks this on the receiver before turning reports on).

A complete report looks exactly like this:

```text
{"type":"event","payload":{"website":"11111111-2222-3333-4444-555555555555","hostname":"vido","url":"/usage-report","name":"weekly_usage","id":"3f2a6c1e-8f0b-4d5e-9a7c-1b2c3d4e5f60","ip":"127.0.0.1","data":{"version":"0.1.2","subtitles_auto_7d":7,"subtitles_embedded_7d":4,"subtitles_online_7d":2,"subtitles_asr_7d":1}}}
```

The settings page shows the last report that was actually sent, byte for byte, and when it was sent.

## What is never sent

- Titles, file names, folder paths, or any other metadata about your library
- What you watch, request or download
- API keys, passwords or any other setting
- Anything that identifies you, your NAS or your network

## Turning it on or off

- **First-run setup** asks once. The pre-selected answer is "off".
- **Settings → Connection** has a switch at the very bottom you can change at any time. After you turn it off, nothing more is sent. Turning it back on reuses the same random ID.

## Where it goes and how often

- To the maintainer's self-hosted [Umami](https://umami.is) analytics instance. Umami does not store IP addresses.
- At most once every 7 days. If a send fails, Vido waits the full 7 days before trying again. A failure is never shown as an error and never affects anything else.
- Builds made from source (and forks) have no receiver configured, so the report shows as unavailable there.
