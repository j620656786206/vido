# Scripts

This directory contains utility scripts for development and setup.

## Available Scripts

### subtitle-blind-eval.py

Builds a blind A/B scoring sheet for two subtitle-translation variants of the same episode, then tallies the scores against the pre-registered thresholds from story `eval-1-translation-blind-eval`.

**Usage:**

```bash
# 1. sample 50 shared cues, shuffle left/right, hide the answer key
python3 scripts/subtitle-blind-eval.py build \
  --source eval/<slug>/source.srt --a eval/<slug>/haiku.srt --b eval/<slug>/sonnet.srt \
  --out eval/<slug> --sample 50 --seed 42

# 2. fill left_score / right_score (0/1/2) in eval/<slug>/sheet.csv

# 3. un-blind and print the verdict
python3 scripts/subtitle-blind-eval.py score eval/<slug>
```

**Requirements:** Python 3.9+, stdlib only.
