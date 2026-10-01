"""Publish reproducible legacy archive names from a canonical release."""
import gzip
import hashlib
import io
import json
import re
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path

version = sys.argv[1]
if not re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", version):
    raise SystemExit("Expected vX.Y.Z")
canonical = "baldaworks/codex-acp"
legacy = "normahq/codex-acp-bridge"

def run(*args):
    return subprocess.check_output(args, text=True).strip()

with tempfile.TemporaryDirectory(prefix="legacy-assets-") as directory:
    root = Path(directory)
    source = root / "canonical"
    source.mkdir()
    run("gh", "release", "download", version, "--repo", canonical, "--dir", str(source))
    sums = dict(line.split("  ", 1)[::-1] for line in (source / "checksums.txt").read_text().splitlines())
    outputs = []
    for target in ["darwin-amd64", "darwin-arm64", "linux-amd64", "linux-arm64", "windows-amd64"]:
        canonical_stem = f"codex-acp-{version}-{target}"
        archive_path = source / (canonical_stem + ".tar.gz")
        if hashlib.sha256(archive_path.read_bytes()).hexdigest() != sums.get("./" + archive_path.name, sums.get(archive_path.name)):
            raise SystemExit("Canonical archive checksum mismatch: " + target)
        binary = "codex-acp.exe" if target.startswith("windows-") else "codex-acp"
        with tarfile.open(archive_path) as archive:
            member = archive.extractfile(canonical_stem + "/" + binary)
            if member is None:
                raise SystemExit("Missing canonical binary")
            binary_bytes = member.read()
        legacy_stem = f"codex-acp-bridge-{version}-{target}"
        output = root / (legacy_stem + ".tar.gz")
        with output.open("wb") as file, gzip.GzipFile(filename="", fileobj=file, mode="wb", mtime=0) as compressed, tarfile.open(fileobj=compressed, mode="w") as archive:
            for name, content, mode in [
                (binary.replace("codex-acp", "codex-acp-bridge", 1), binary_bytes, 0o755),
                ("README.md", run("git", "show", f"{version}:README.md").encode() + b"\n", 0o644),
                ("LICENSE", run("git", "show", f"{version}:LICENSE").encode() + b"\n", 0o644),
            ]:
                info = tarfile.TarInfo(legacy_stem + "/" + name)
                info.size, info.mode = len(content), mode
                archive.addfile(info, io.BytesIO(content))
        outputs.append(output)
    checksum_file = root / "checksums.txt"
    checksum_file.write_text("".join(f"{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}\n" for p in outputs))
    outputs.append(checksum_file)
    # Distinguish a missing release from authentication or network errors.
    releases = json.loads(run("gh", "api", f"repos/{legacy}/releases?per_page=100"))
    release = next((r for r in releases if r["tag_name"] == version), None)
    if release is None:
        subprocess.run(["gh", "release", "create", version, "--repo", legacy, "--verify-tag", "--title", version,
                        "--notes", f"Legacy compatibility for https://github.com/{canonical}/releases/tag/{version}. Native binaries are shared with the canonical release."], check=True)
        existing = set()
    else:
        existing = {asset["name"] for asset in release["assets"]}
    for artifact in outputs:
        if artifact.name in existing:
            previous = root / "existing"
            previous.mkdir(exist_ok=True)
            run("gh", "release", "download", version, "--repo", legacy, "--pattern", artifact.name, "--dir", str(previous))
            if (previous / artifact.name).read_bytes() != artifact.read_bytes():
                raise SystemExit("Refusing to replace accepted legacy asset: " + artifact.name)
        else:
            subprocess.run(["gh", "release", "upload", version, str(artifact), "--repo", legacy], check=True)
    print("Legacy archive assets verified for " + version)
