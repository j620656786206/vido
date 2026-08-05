import requests

BASE_URL = "http://localhost:8090"
TIMEOUT = 30
HEADERS = {"Content-Type": "application/json"}


def test_subtitle_pipeline_run_gated_409_when_unconfigured():
    url = f"{BASE_URL}/api/v1/subtitles/pipeline/run"

    # Case 1: Valid-shaped body expecting 409 AI_NOT_CONFIGURED
    payload_valid = {"media_id": "1", "media_type": "movie"}
    resp = requests.post(url, json=payload_valid, headers=HEADERS, timeout=TIMEOUT)
    try:
        assert resp.status_code == 409, f"Expected 409, got {resp.status_code}"
        j = resp.json()
        assert j.get("success") is False, "Expected success:false in response"
        err = j.get("error")
        assert err and isinstance(err, dict), "Expected 'error' dict in response"
        assert err.get("code") == "AI_NOT_CONFIGURED", f"Expected error code 'AI_NOT_CONFIGURED', got {err.get('code')}"
        msg = err.get("message")
        suggestion = err.get("suggestion")
        # Check that message and suggestion are present, non-empty, and contain Chinese characters (basic check for non-empty)
        assert msg and isinstance(msg, str) and len(msg.strip()) > 0, "Expected non-empty Chinese message"
        assert suggestion and isinstance(suggestion, str) and len(suggestion.strip()) > 0, "Expected non-empty Chinese suggestion"
    except Exception:
        # Print response content on failure to aid debugging
        print("Response content (valid case):", resp.text)
        raise

    # Case 2: Missing media_type expecting 400 VALIDATION_INVALID_FORMAT
    payload_missing_media_type = {"media_id": "1"}
    resp2 = requests.post(url, json=payload_missing_media_type, headers=HEADERS, timeout=TIMEOUT)
    try:
        assert resp2.status_code == 400, f"Expected 400, got {resp2.status_code}"
        j2 = resp2.json()
        assert j2.get("success") is False, "Expected success:false in response"
        err2 = j2.get("error")
        assert err2 and isinstance(err2, dict), "Expected 'error' dict in response"
        assert err2.get("code") == "VALIDATION_INVALID_FORMAT", f"Expected error code 'VALIDATION_INVALID_FORMAT', got {err2.get('code')}"
    except Exception:
        print("Response content (missing media_type case):", resp2.text)
        raise


test_subtitle_pipeline_run_gated_409_when_unconfigured()