---
name: Let user demo before proposing solutions
description: When the user offers to demo a bug, wait for their description and demo before locking in problem framing — don't anchor on the bug title literally
type: feedback
originSessionId: 423a478c-28cc-4e1b-8e8f-dd71fc25846c
---
When investigating a bug the user is about to demo, do NOT propose solutions based on the literal title or sprint-status one-liner. The user often has a more nuanced understanding of the actual user pain that won't surface until they describe what they perceive as wrong.

**Why:** During bugfix-10-4-hover-preview-viewport-flip discussion (2026-05-08), the title and backlog entry framed it as "preview floats below card, gets clipped/forces scroll" — a viewport-edge problem. I locked in on that framing and proposed Radix Popover with collision avoidance. After the user demoed, the real issue turned out to be *perceptual proximity* — preview appears outside the user's focal area so hover feels unresponsive entirely, regardless of viewport position. That reframing changed the right answer from "Radix Popover" to "Netflix-style scale-up overlay". Two rounds of mis-aligned proposals before the user finally reframed.

**How to apply:** Before launching local dev / waiting for a demo, propose at most a *menu of possible interpretations*, not a single recommendation. Save the actual recommendation for after the user has shown and described what they perceive as broken. When the user says "I'll show you" — that's a signal that the literal bug title is incomplete; wait for their narrative.
