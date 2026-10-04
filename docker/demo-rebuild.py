#!/usr/bin/env python3
"""Prepare a separate demo from a verified release; never switch running services."""
import argparse
import gzip
import hashlib
import ipaddress
import json
import os
import re
import shlex
import shutil
import stat
import subprocess
import sys
import tarfile
import uuid
from datetime import datetime, timezone
from pathlib import Path, PurePosixPath

MAX_FILES = 10000
MAX_BYTES = 1024 * 1024 * 1024
MAX_PATH_BYTES = 4096
MAX_PATH_DEPTH = 64
COPY_BYTES = 64 * 1024
MAX_METADATA_BYTES = 16 * 1024 * 1024
MAX_METADATA_READ = 1024 * 1024
HOST_OVERRIDES = ("COMPOSE_PROJECT_NAME", "DEMO_PROJECT_NAME", "DEPLOY_ROOT")
SERVICES = "web api ai-rpc rpc mysql redis etcd"
REQUIRED = (
    "backend/docker/demo.sh", "backend/docker/demo-bootstrap.py",
    "backend/docker/demo-compose.yml", "backend/docker/deploy-compose.yml",
    "backend/docker/Caddyfile.demo", "backend/test/sh/db.sh",
    "backend/test/sh/mysql-client.sh", "backend/bin/runtime/Dockerfile",
    "backend/bin/runtime/applet-api", "backend/bin/runtime/applet-rpc",
    "backend/bin/runtime/applet-ai-rpc", "frontend/dist/index.html",
)


class ReleaseStream:
    """Bound tar parser allocations before it exposes extended-header members."""

    def __init__(self, source):
        self.decoded = gzip.GzipFile(fileobj=source, mode="rb")
        self.check_metadata = True
        self.metadata_bytes = 0

    def __enter__(self):
        return self

    def __exit__(self, *unused):
        self.decoded.close()

    def tell(self):
        return self.decoded.tell()

    @staticmethod
    def max_offset():
        # Include tar headers, file padding and final record padding.
        return MAX_BYTES + MAX_METADATA_BYTES + MAX_FILES * 512 + 10240

    def read(self, size=-1):
        if size < 0 or size > MAX_METADATA_READ:
            raise ValueError("Release metadata read exceeds the size limit")
        if self.check_metadata and self.metadata_bytes + size > MAX_METADATA_BYTES:
            raise ValueError("Release metadata exceeds the size limit")
        if self.tell() + size > self.max_offset():
            raise ValueError("Release tar stream exceeds the size limit")
        content = self.decoded.read(size)
        if self.check_metadata:
            self.metadata_bytes += len(content)
        return content

    def seek(self, offset, whence=os.SEEK_SET):
        if whence == os.SEEK_SET:
            position = offset
        elif whence == os.SEEK_CUR:
            position = self.tell() + offset
        else:
            raise ValueError("Release seeks from end are prohibited")
        if position < 0 or position > self.max_offset():
            raise ValueError("Release tar stream exceeds the size limit")
        return self.decoded.seek(offset, whence)


def selected_path(value, root=None, directory=False):
    path = Path(value)
    if not path.is_absolute():
        raise ValueError("Use absolute paths")
    # Reject existing symlinks, including parent links, before resolving paths.
    for item in (path, *path.parents):
        if item.is_symlink():
            raise ValueError("Deployment input paths must not contain symbolic links")
    path = path.resolve(strict=True)
    if path == Path(path.anchor):
        raise ValueError("A filesystem root cannot be a deployment directory")
    if root is not None and not path.is_relative_to(root):
        raise ValueError("All inputs must stay inside the selected deployment directory")
    if directory:
        if not path.is_dir():
            raise ValueError("Expected an existing directory")
    elif not path.is_file():
        raise ValueError("Expected a regular file")
    return path


def members_checked(archive):
    members = []
    paths = {}
    total = 0
    for member in archive:
        members.append(member)
        if len(members) > MAX_FILES:
            raise ValueError("Unexpected release file count")
        name = member.name.rstrip("/")
        pieces = name.split("/")
        if (not name or any(part in ("", ".", "..") for part in pieces)
                or "\\" in name or ":" in name or name.startswith("/")
                or any(ord(char) < 32 for char in name)
                or len(name.encode("utf-8")) > MAX_PATH_BYTES
                or len(pieces) > MAX_PATH_DEPTH
                or any(len(part.encode("utf-8")) > 255 for part in pieces)
                or pieces[0] not in ("backend", "frontend")):
            raise ValueError("Unsafe or unexpected release path")
        if not (member.isdir() or member.isfile()):
            raise ValueError("Release links, devices and special files are prohibited")
        if name in paths:
            raise ValueError("Duplicate release paths are prohibited")
        if any(part.startswith(".env") or part in (".git", "node_modules")
               or (part == "runtime" and (index != 2 or pieces[:2] != ["backend", "bin"]))
               for index, part in enumerate(pieces)):
            raise ValueError("Release contains private settings or unexpected runtime data")
        if (member.size < 0 or (member.isdir() and member.size != 0)
                or member.sparse is not None or (member.isfile() and len(pieces) == 1)):
            raise ValueError("Invalid release entry")
        paths[name] = member
        total += member.size
        if total > MAX_BYTES:
            raise ValueError("Release exceeds the extraction size limit")
    if not members:
        raise ValueError("Unexpected release file count")
    for name in paths:
        for parent in PurePosixPath(name).parents:
            if str(parent) in paths and not paths[str(parent)].isdir():
                raise ValueError("A release file cannot be a parent directory")
    if any(name not in paths or not paths[name].isfile() for name in REQUIRED):
        raise ValueError("Release is missing a required deployment file")
    with archive.extractfile(paths["backend/docker/demo-compose.yml"]) as source:
        compose_text = source.read(65537)
    if (len(compose_text) > 65536 or not re.search(
            rb"(?m)^name:\s*\$\{DEMO_PROJECT_NAME:-go-zero-admin-demo\}\s*$", compose_text)):
        raise ValueError("Release is too old: isolated DEMO_PROJECT_NAME support is required")
    return members


def extract_checked(archive, members, destination):
    # Explicit regular-file writes avoid tar extractall behavior differences.
    remaining_budget = MAX_BYTES
    for member in members:
        target = destination.joinpath(*member.name.rstrip("/").split("/"))
        if not target.resolve().is_relative_to(destination.resolve()):
            raise ValueError("Extraction target escaped the fresh release")
        if member.isdir():
            target.mkdir(mode=0o755, parents=True, exist_ok=True)
            continue
        if member.size < 0 or member.size > remaining_budget:
            raise ValueError("Release exceeds the extraction size limit")
        target.parent.mkdir(mode=0o755, parents=True, exist_ok=True)
        with archive.extractfile(member) as source, target.open("xb") as output:
            remaining = member.size
            while remaining:
                chunk = source.read(min(COPY_BYTES, remaining))
                if not chunk or len(chunk) > remaining:
                    raise ValueError("Release file length differs from its declaration")
                output.write(chunk)
                remaining -= len(chunk)
            if source.read(1):
                raise ValueError("Release file length differs from its declaration")
        remaining_budget -= member.size
        target.chmod(0o644 | (member.mode & 0o111))


def previous_deployment(backend, root):
    backend = selected_path(backend, root, directory=True)
    if backend.name != "backend" or backend.parent.parent.name != "releases":
        raise ValueError("Previous backend must be ROOT/releases/RELEASE/backend")
    deployment = backend.parents[2]
    if not deployment.is_relative_to(root):
        raise ValueError("Previous deployment escaped the selected root")
    for relative in ("docker/demo-compose.yml", "docker/demo.sh"):
        selected_path(str(backend / relative), root)
    return backend, deployment


def copy_tls(source, destination, root):
    source = selected_path(str(source), root, directory=True)
    for folder, subdirs, files in os.walk(source, followlinks=False):
        for name in (*subdirs, *files):
            item = Path(folder) / name
            if item.is_symlink() or not (item.is_dir() or stat.S_ISREG(item.stat().st_mode)):
                raise ValueError("TLS state contains a link or special file; reuse refused")
    # Caddy owns certificate data. Keep this copy private and outside release tar.
    shutil.copytree(source, destination, dirs_exist_ok=True)
    for folder, subdirs, files in os.walk(destination):
        Path(folder).chmod(0o700)
        for name in files:
            (Path(folder) / name).chmod(0o600)


def shell_command(backend, command):
    return (f"(cd {shlex.quote(str(backend))} && unset {' '.join(HOST_OVERRIDES)} "
            f"&& {command})")


def compose_command(backend, arguments):
    return shell_command(backend, "docker compose --env-file docker/.env.deploy "
                         f"-f docker/demo-compose.yml {arguments}")


def verify_new_environment(env_file, deployment, project):
    # Inspect only the newly prepared env. Never read the previous deployment.
    target = env_file.resolve(strict=True)
    expected = deployment / "runtime/.env.demo"
    if target != expected or not target.is_relative_to(deployment):
        raise ValueError("New environment must target its own runtime/.env.demo")
    selected_path(str(target), deployment)
    lines = target.read_text(encoding="utf-8").splitlines()
    for key, expected_value in (("DEMO_PROJECT_NAME", project), ("DEPLOY_ROOT", str(deployment))):
        matching = [line for line in lines if line.startswith(key + "=")]
        if matching != [key + "=" + expected_value]:
            raise ValueError("Release is too old or misconfigured: isolated deployment settings were not persisted")


def commands_text(backend, previous, domain, address):
    stop_new = compose_command(backend, "stop " + SERVICES)
    stop_old = compose_command(previous, "stop " + SERVICES) if previous else "# Provide --previous-backend to generate the old project's stop command."
    old_up = compose_command(previous, "up -d mysql redis etcd rpc ai-rpc api web") if previous else "# Return to the previous backend and start its existing seven services."
    backup_commands = (compose_command(previous, "stop web api ai-rpc rpc") + " &&\n"
                       + shell_command(previous, "sh docker/demo.sh backup")) if previous else "# If the old database is readable, stop its write services and run sh docker/demo.sh backup first."
    new_cd = shlex.quote(str(backend))
    switch_commands = ((stop_old + " &&\n") if previous else (stop_old + "\n")) + (
        f"cd {new_cd} &&\nsh docker/demo.sh init &&\n"
        "sh docker/demo.sh start &&\nsh docker/demo.sh status")
    rollback_commands = stop_new + (" &&\n" if previous else "\n") + old_up
    return f"""# 已准备的新环境：切换命令

本文件由工具生成。工具未执行 Docker、数据库迁移、停机或入口切换。旧数据库和旧配置均保留。
新环境从发布包初始化空数据库，保留公开只读 `admin / 123456` 和私有随机维护账号；AI 不配置 Key。
命令先清除宿主机残留的项目名和数据目录变量，再使用各环境自己的 `.env.deploy`。

先确认发布包和新配置：

```sh
cd {new_cd}
unset COMPOSE_PROJECT_NAME DEMO_PROJECT_NAME DEPLOY_ROOT
docker compose --env-file docker/.env.deploy -f docker/demo-compose.yml config --quiet
```

旧 MySQL 仍可用时，可先停止旧写入服务并备份。坏库备份失败不阻止选择重建空库；备份只保存在旧发布目录，随后人工转存。

```sh
{backup_commands}
```

切换会短暂停机。2 GB 服务器避免两套同时运行：先停止旧项目全部七个服务，再初始化新项目。

```sh
unset COMPOSE_PROJECT_NAME DEMO_PROJECT_NAME DEPLOY_ROOT
{switch_commands}
```

`init` 使用服务器已缓存的 Docker 层构建三份运行镜像，无需安装 Go 或 Node；初始化、迁移、服务启动通常需要几分钟，首次下载镜像更久。
通过 https://{domain}/ 和 http://{address}/ 测试验证码、正常登录和菜单。完成后可更新项目根目录的 current 符号链接；不要覆盖真实目录。

如新部署失败，先停止新项目，再启动旧项目，恢复原数据库和原配置：

```sh
{rollback_commands}
```

只使用 `stop`；不要执行 `down -v`、删除、搬移或复用旧 `runtime/mysql`。这里的快速重建会生成新系统，不恢复旧业务数据。
"""


def prepare_rebuild(args):
    root = selected_path(args.root, directory=True)
    release_tar = selected_path(args.archive, root)
    if not re.fullmatch(r"[a-fA-F0-9]{64}", args.sha256):
        raise ValueError("Provide the trusted release SHA256")
    if not re.fullmatch(r"[a-zA-Z0-9](?:[a-zA-Z0-9.-]*[a-zA-Z0-9])?", args.domain) or "." not in args.domain:
        raise ValueError("Invalid domain")
    ipaddress.IPv4Address(args.ip)
    previous = previous_deployment(args.previous_backend, root) if args.previous_backend else None
    if args.reuse_tls and previous is None:
        raise ValueError("--reuse-tls requires --previous-backend")
    rebuilds = root / "rebuilds"
    if rebuilds.is_symlink() or (rebuilds.exists() and not rebuilds.is_dir()):
        raise ValueError("The rebuilds path must be a regular directory")
    if rebuilds.exists() and not rebuilds.resolve().is_relative_to(root):
        raise ValueError("The rebuilds path escaped the selected root")
    identifier = datetime.now(timezone.utc).strftime("%Y%m%d-%H%M%S") + "-" + uuid.uuid4().hex[:12]
    project = "go-zero-admin-demo-rebuild-" + identifier
    deployment = rebuilds / identifier
    release = deployment / "releases" / identifier
    staging = release.with_name(identifier + ".staging")
    # Verify before creating anything. Keep the same open file for extraction.
    with release_tar.open("rb") as source:
        digest = hashlib.file_digest(source, "sha256").hexdigest()
        if digest != args.sha256.lower():
            raise ValueError("SHA256 mismatch; no rebuild was created")
        source.seek(0)
        with ReleaseStream(source) as decoded:
            with tarfile.open(fileobj=decoded, mode="r:") as archive:
                members = members_checked(archive)
                decoded.check_metadata = False
                rebuilds.mkdir(mode=0o700, exist_ok=True)
                deployment.mkdir(mode=0o700)
                staging.mkdir(mode=0o700, parents=True)
                extract_checked(archive, members, staging)
    staging.rename(release)
    backend = release / "backend"
    environment = os.environ.copy()
    for name in HOST_OVERRIDES:
        environment.pop(name, None)
    environment["DEMO_PROJECT_NAME"] = project
    result = subprocess.run([sys.executable, str(backend / "docker/demo-bootstrap.py"),
                             "prepare", str(deployment), args.domain, args.ip], cwd=backend,
                            env=environment, capture_output=True, text=True, check=False)
    if result.returncode:
        raise RuntimeError(f"Preparation failed; fresh files retained at {deployment}. Old deployment was untouched.")
    verify_new_environment(backend / "docker/.env.deploy", deployment, project)
    if args.reuse_tls:
        copy_tls(previous[1] / "runtime/caddy-data", deployment / "runtime/caddy-data", root)
    manifest = {
        "prepared_at": datetime.now(timezone.utc).isoformat(), "deployment_root": str(deployment),
        "backend": str(backend), "compose_project": project, "sha256": digest,
        "previous_backend": str(previous[0]) if previous else None,
        "services_switched": False, "tls_reused": bool(args.reuse_tls),
    }
    (deployment / "rebuild.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    instructions = deployment / "RECOVERY_COMMANDS.md"
    instructions.write_text(commands_text(backend, previous[0] if previous else None, args.domain, args.ip), encoding="utf-8")
    print(f"Fresh deployment prepared: {deployment}")
    print(f"Review switch and rollback commands: {instructions}")
    print("Existing services and data were not changed. No Docker command was executed.")
    return deployment


def parser():
    command = argparse.ArgumentParser(description=__doc__)
    command.add_argument("root", help="Existing authorized absolute deployment root")
    command.add_argument("archive", help="Existing release tar.gz inside the root")
    command.add_argument("sha256", help="Expected SHA256 from a trusted package summary")
    command.add_argument("domain")
    command.add_argument("ip")
    command.add_argument("--previous-backend", help="Old ROOT/releases/RELEASE/backend; never reads its env")
    command.add_argument("--reuse-tls", action="store_true", help="Copy previous Caddy certificate state into the new private runtime")
    return command


if __name__ == "__main__":
    try:
        prepare_rebuild(parser().parse_args())
    except (ValueError, RuntimeError, OSError, EOFError, tarfile.TarError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
