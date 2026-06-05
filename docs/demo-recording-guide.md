# Frontend Demo Recording Guide

This document is presenter-facing. Keep these cues out of the product page so the workspace itself stays focused on novel authors and screenplay drafting.

## Default walkthrough

Recommended opening sample:
- `职场`

Recommended first run:
- keep `generationMode=deterministic`

Suggested 90-second flow:
1. Start from the left workspace and explain the source title, adaptation style, audience, notes, and multi-chapter input.
2. Click the primary generate action and let the middle column show the real status polling plus stage progression.
3. Move to the right workspace and explain the YAML draft first, then the structured summary, then the reset / copy / export actions.

## Alternate samples

Use these when you want a different tone in the same live chain:
- `悬疑`: night return, clue discovery, active investigation
- `校园运动`: relay pressure, team conflict, growth under deadline

## What to verify on camera

- Sample preset switching works before submission
- A real job is created from the frontend form
- The status column moves through queued / running / succeeded
- The result area loads backend-returned YAML and structured summary
- Local YAML edits can be reset or exported
- Failed jobs can be regenerated from the current form state

## Presenter note

If you need technical setup, port contracts, or smoke-check commands, use:
- [`../README.md`](../README.md)
- [`implementation-progress.md`](implementation-progress.md)
- [`frontend.md`](frontend.md)
