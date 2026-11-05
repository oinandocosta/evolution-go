"""Block publication unless the fixed Docker Hub repository is already public."""

import json
import sys
from urllib.error import HTTPError, URLError
from urllib.request import HTTPRedirectHandler, Request, build_opener


REPOSITORY_URL = "https://hub.docker.com/v2/namespaces/nandocosta/repositories/evolution-go"
TIMEOUT_SECONDS = 15
MAX_RESPONSE_BYTES = 131072


class GuardError(Exception):
    pass


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise GuardError("Docker Hub returned a redirect; publication is blocked.")


def request_json(opener, request):
    try:
        with opener.open(request, timeout=TIMEOUT_SECONDS) as response:
            if response.status != 200:
                raise GuardError("Docker Hub did not return HTTP 200; publication is blocked.")
            body = response.read(MAX_RESPONSE_BYTES + 1)
        if len(body) > MAX_RESPONSE_BYTES:
            raise GuardError("Docker Hub response exceeded the limit; publication is blocked.")
        payload = json.loads(body)
        if not isinstance(payload, dict):
            raise GuardError("Docker Hub returned invalid metadata; publication is blocked.")
        return payload
    except HTTPError as error:
        raise GuardError(f"Docker Hub returned HTTP {error.code}; publication is blocked.") from None
    except (URLError, TimeoutError, OSError):
        raise GuardError("Docker Hub connection failed or timed out; publication is blocked.") from None
    except (ValueError, UnicodeError):
        raise GuardError("Docker Hub returned invalid JSON; publication is blocked.") from None


def verify_public_repository(opener=None):
    if opener is None:
        opener = build_opener(NoRedirect())
    repository_request = Request(REPOSITORY_URL, headers={"Accept": "application/json"}, method="GET")
    repository = request_json(opener, repository_request)
    if repository.get("is_private") is not False:
        raise GuardError("nandocosta/evolution-go must already exist as Public; publication is blocked.")


def main():
    try:
        verify_public_repository()
    except GuardError as error:
        print(str(error), file=sys.stderr)
        return 1
    print("Verified existing public repository: nandocosta/evolution-go")
    return 0


if __name__ == "__main__":
    sys.exit(main())
