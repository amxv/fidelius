"""Submit two synthetic values to each compiled X11 frontend under Xvfb."""
import os
from pathlib import Path
import subprocess
import sys
import time


def xdo(*args):
    return subprocess.check_output(["xdotool", *args], stderr=subprocess.DEVNULL)


def exercise(executable):
    reader, writer = os.pipe()
    with os.fdopen(reader, "rb") as response:
        process = subprocess.Popen(
            ["sh", "-c", f'exec "$1" 3>&{writer}', "smoke", str(executable)],
            pass_fds=(writer,), stdin=subprocess.PIPE, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE, env={**os.environ, "GDK_BACKEND": "x11", "QT_QPA_PLATFORM": "xcb"},
        )
        os.close(writer)
        try:
            process.stdin.write(b"Test request\0Expires shortly.\0FIRST\0SECOND\0")
            process.stdin.flush()
            process.stdin.close()
            for _ in range(100):
                if process.poll() is not None:
                    raise RuntimeError(f"{executable.name} exited before showing the form")
                try:
                    window = xdo("search", "--onlyvisible", "--name", "^Fidelius$").splitlines()[0]
                    break
                except (subprocess.CalledProcessError, IndexError):
                    time.sleep(0.1)
            else:
                raise RuntimeError("window did not appear")
            xdo("windowfocus", "--sync", window)
            xdo("type", "example-first")
            xdo("key", "Return")
            xdo("type", "example-second")
            xdo("key", "Return")
            status = process.wait(timeout=10)
            if status:
                raise RuntimeError(f"{executable.name} exited with {status}")
            assert response.read() == b"Rexample-first\0example-second\0"
            assert not process.stdout.read()
            assert b"example-" not in process.stderr.read()
        finally:
            if process.poll() is None:
                process.kill()
                process.wait()


for helper in ("gtk", "qt"):
    exercise(Path(sys.argv[1]).resolve() / helper)
