#!/usr/bin/env python3
from __future__ import annotations

import argparse
import functools
import os
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer


class QuietHandler(SimpleHTTPRequestHandler):
    def log_message(self, format: str, *args) -> None:
        return


def main() -> None:
    parser = argparse.ArgumentParser(description="Quiet static server for LAN client assets.")
    parser.add_argument("--bind", default="0.0.0.0")
    parser.add_argument("--port", type=int, required=True)
    parser.add_argument("--directory", default=os.getcwd())
    args = parser.parse_args()

    handler = functools.partial(QuietHandler, directory=args.directory)
    server = ThreadingHTTPServer((args.bind, args.port), handler)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
