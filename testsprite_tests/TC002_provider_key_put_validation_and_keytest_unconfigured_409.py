import requests

BASE_URL = "http://localhost:8090"
HEADERS_JSON = {"Content-Type": "application/json"}
TIMEOUT = 30

def test_provider_key_put_validation_and_keytest_unconfigured_409():
    # Step 1: PUT /api/v1/settings/keys with empty JSON {}
    url_keys = f"{BASE_URL}/api/v1/settings/keys"
    try:
        resp1 = requests.put(url_keys, json={}, headers=HEADERS_JSON, timeout=TIMEOUT)
    except requests.RequestException as e:
        assert False, f"Step 1 request failed: {e}"
    assert resp1.status_code == 400, f"Expected 400 but got {resp1.status_code} in Step 1"
    try:
        json1 = resp1.json()
    except ValueError:
        assert False, "Step 1 response is not JSON"
    assert json1.get("success") is False, "Step 1 success flag should be false"
    error1 = json1.get("error", {})
    assert error1.get("code") == "VALIDATION_REQUIRED_FIELD", f"Step 1 error code expected VALIDATION_REQUIRED_FIELD but got {error1.get('code')}"

    # Step 2: PUT /api/v1/settings/keys with malformed body (bare string)
    # Sending data as plain text rather than JSON to simulate malformed
    malformed_body = "malformed-string-not-json"
    try:
        resp2 = requests.put(url_keys, data=malformed_body, headers={"Content-Type": "text/plain"}, timeout=TIMEOUT)
    except requests.RequestException as e:
        assert False, f"Step 2 request failed: {e}"
    assert resp2.status_code == 400, f"Expected 400 but got {resp2.status_code} in Step 2"
    try:
        json2 = resp2.json()
    except ValueError:
        assert False, "Step 2 response is not JSON"
    assert json2.get("success") is False, "Step 2 success flag should be false"
    error2 = json2.get("error", {})
    assert error2.get("code") == "VALIDATION_INVALID_FORMAT", f"Step 2 error code expected VALIDATION_INVALID_FORMAT but got {error2.get('code')}"

    # Step 3: POST /api/v1/settings/keys/test with no body (no key configured)
    url_test = f"{BASE_URL}/api/v1/settings/keys/test"
    try:
        resp3 = requests.post(url_test, headers=HEADERS_JSON, timeout=TIMEOUT)
    except requests.RequestException as e:
        assert False, f"Step 3 request failed: {e}"
    assert resp3.status_code == 409, f"Expected 409 but got {resp3.status_code} in Step 3"
    try:
        json3 = resp3.json()
    except ValueError:
        assert False, "Step 3 response is not JSON"
    assert json3.get("success") is False, "Step 3 success flag should be false"
    error3 = json3.get("error", {})
    assert error3.get("code") == "AI_NOT_CONFIGURED", f"Step 3 error code expected AI_NOT_CONFIGURED but got {error3.get('code')}"
    message3 = error3.get("message", "")
    assert "尚未設定" in message3, f"Step 3 error message expected to mention '尚未設定' but got: {message3}"

test_provider_key_put_validation_and_keytest_unconfigured_409()