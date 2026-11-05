import importlib.util
import io
import json
from pathlib import Path
import unittest
from contextlib import redirect_stderr
from unittest.mock import Mock, patch
from urllib.error import HTTPError, URLError


spec = importlib.util.spec_from_file_location("public_repository", Path(__file__).with_name("check-public-repository.py"))
guard = importlib.util.module_from_spec(spec)
spec.loader.exec_module(guard)


class FakeResponse(io.BytesIO):
    status = 200


def response(payload):
    return FakeResponse(json.dumps(payload).encode())


class PublicRepositoryTests(unittest.TestCase):
    def test_public_repository_passes_using_only_one_anonymous_get(self):
        opener = Mock(open=Mock(return_value=response({"is_private": False})))
        guard.verify_public_repository(opener)
        opener.open.assert_called_once()
        call = opener.open.call_args
        request = call.args[0]
        self.assertEqual(request.full_url, guard.REPOSITORY_URL)
        self.assertEqual(request.method, "GET")
        self.assertIsNone(request.data)
        self.assertFalse(request.has_header("Authorization"))
        self.assertFalse(request.has_header("Cookie"))
        self.assertEqual(call.kwargs, {"timeout": 15})

    def test_private_missing_or_non_boolean_privacy_is_blocked(self):
        for metadata in ({"is_private": True}, {}, {"is_private": "false"}, {"is_private": 0}, {"is_private": None}):
            opener = Mock(open=Mock(return_value=response(metadata)))
            with self.subTest(metadata=metadata), self.assertRaises(guard.GuardError):
                guard.verify_public_repository(opener)

    def test_http_errors_and_connection_failures_block_publication(self):
        for failure in [
            *(HTTPError(guard.REPOSITORY_URL, status, "response-detail-must-not-print", {}, None) for status in (301, 401, 403, 404, 500)),
            URLError("response-detail-network"), TimeoutError("response-detail-timeout"),
        ]:
            opener = Mock(open=Mock(side_effect=failure))
            with self.subTest(failure=type(failure).__name__), self.assertRaises(guard.GuardError) as raised:
                guard.verify_public_repository(opener)
            self.assertNotIn("response-detail", str(raised.exception))

    def test_invalid_json_or_non_object_metadata_blocks_publication(self):
        for body in (b"not-json", b"[]", b"null", b"\xff", b"x" * (guard.MAX_RESPONSE_BYTES + 1)):
            opener = Mock(open=Mock(return_value=FakeResponse(body)))
            with self.subTest(size=len(body)), self.assertRaises(guard.GuardError):
                guard.verify_public_repository(opener)

    def test_non_200_response_is_blocked(self):
        rejected = response({"is_private": False})
        rejected.status = 204
        opener = Mock(open=Mock(return_value=rejected))
        with self.assertRaises(guard.GuardError):
            guard.verify_public_repository(opener)

    def test_redirect_handler_never_forwards_request(self):
        with self.assertRaises(guard.GuardError):
            guard.NoRedirect().redirect_request(None, None, 302, "redirect", {}, "https://example.invalid/")

    def test_main_reports_failure_without_dumping_response_details(self):
        output = io.StringIO()
        with patch.object(guard, "verify_public_repository", side_effect=guard.GuardError("publication blocked")), redirect_stderr(output):
            self.assertEqual(guard.main(), 1)
        self.assertEqual(output.getvalue(), "publication blocked\n")


if __name__ == "__main__":
    unittest.main()
