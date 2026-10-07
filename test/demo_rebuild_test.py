"""Archive boundary and prepare-only recovery checks; no real Docker or database."""
import hashlib
import importlib.util
import io
import json
import subprocess
import tarfile
import tempfile
import unittest
from argparse import Namespace
from pathlib import Path
from unittest.mock import patch

MODULE_PATH = Path(__file__).resolve().parents[1] / "docker/demo-rebuild.py"
SPEC = importlib.util.spec_from_file_location("demo_rebuild", MODULE_PATH)
REBUILD = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(REBUILD)


class DemoRebuildTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary.name).resolve()
        self.addCleanup(self.temporary.cleanup)

    def archive(self, extras=(), omitted=(), contents=None):
        destination = self.root / "release.tar.gz"
        with tarfile.open(destination, "w:gz") as target:
            for name in (*REBUILD.REQUIRED, "backend/data/db/migrations/fixture.sql"):
                if name not in omitted:
                    entry = tarfile.TarInfo(name)
                    content = b"safe fixture"
                    if name == "backend/docker/demo-compose.yml":
                        content = b"name: ${DEMO_PROJECT_NAME:-go-zero-admin-demo}\nservices: {}\n"
                    if contents and name in contents:
                        content = contents[name]
                    entry.size = len(content)
                    target.addfile(entry, io.BytesIO(content))
            for entry, content in extras:
                target.addfile(entry, io.BytesIO(content) if content is not None else None)
        digest = hashlib.sha256(destination.read_bytes()).hexdigest()
        return destination, digest

    def arguments(self, archive=None, digest=None, **extra):
        if archive is None:
            archive, digest = self.archive()
        values = dict(root=str(self.root), archive=str(archive), sha256=digest,
                      domain="example.com", ip="192.0.2.1", previous_backend=None, reuse_tls=False)
        values.update(extra)
        return Namespace(**values)

    def test_prepares_unique_roots_without_docker_or_old_data_change(self):
        old = self.root / "releases/old/backend"
        (old / "docker").mkdir(parents=True)
        for name in ("demo-compose.yml", "demo.sh"):
            (old / "docker" / name).write_text("old source", encoding="utf-8")
        data = self.root / "runtime/mysql/do-not-touch"
        data.parent.mkdir(parents=True)
        data.write_bytes(b"original database")
        options = self.arguments(previous_backend=str(old))
        with patch.object(REBUILD.subprocess, "run", return_value=subprocess.CompletedProcess([], 0)) as run:
            with patch.object(REBUILD, "verify_new_environment") as verify, patch("builtins.print"), patch.dict(
                    REBUILD.os.environ, {"COMPOSE_PROJECT_NAME": "old-project", "DEMO_PROJECT_NAME": "old-demo", "DEPLOY_ROOT": str(self.root)}):
                first = REBUILD.prepare_rebuild(options)
                second = REBUILD.prepare_rebuild(options)
        self.assertNotEqual(first, second)
        self.assertEqual(data.read_bytes(), b"original database")
        self.assertEqual(run.call_count, 2)
        self.assertEqual(verify.call_count, 2)
        for call in run.call_args_list:
            self.assertEqual(call.args[0][2], "prepare")
            self.assertNotIn("docker", call.args[0])
            self.assertTrue(call.kwargs["env"]["DEMO_PROJECT_NAME"].startswith("go-zero-admin-demo-rebuild-"))
            self.assertNotIn("COMPOSE_PROJECT_NAME", call.kwargs["env"])
            self.assertNotIn("DEPLOY_ROOT", call.kwargs["env"])
        manifest = json.loads((first / "rebuild.json").read_text(encoding="utf-8"))
        self.assertFalse(manifest["services_switched"])
        self.assertEqual(manifest["sha256"], options.sha256)
        commands = (first / "RECOVERY_COMMANDS.md").read_text(encoding="utf-8")
        self.assertIn("stop web api ai-rpc rpc mysql redis etcd", commands)
        self.assertIn("up -d mysql redis etcd rpc ai-rpc api web", commands)
        self.assertGreaterEqual(commands.count("unset COMPOSE_PROJECT_NAME DEMO_PROJECT_NAME DEPLOY_ROOT"), 5)
        self.assertTrue((first / "releases" / first.name / "frontend/dist/index.html").is_file())

    def test_generated_backup_switch_and_rollback_stop_after_errors(self):
        backend = self.root / "new/backend"
        previous = self.root / "old backend"
        commands = REBUILD.commands_text(backend, previous, "example.com", "192.0.2.1")
        backup_line = next(line for line in commands.splitlines() if "sh docker/demo.sh backup)" in line)
        self.assertIn("unset COMPOSE_PROJECT_NAME DEMO_PROJECT_NAME DEPLOY_ROOT", backup_line)
        self.assertIn("stop web api ai-rpc rpc) &&\n" + backup_line, commands)
        self.assertIn(REBUILD.compose_command(previous, "stop " + REBUILD.SERVICES) + " &&\n", commands)
        self.assertIn("sh docker/demo.sh init &&\nsh docker/demo.sh start &&\n", commands)
        self.assertIn(REBUILD.compose_command(backend, "stop " + REBUILD.SERVICES) + " &&\n", commands)
        self.assertIn("'" + str(previous) + "'", backup_line)

    def test_bad_hash_creates_nothing(self):
        options = self.arguments()
        options.sha256 = "0" * 64
        with self.assertRaisesRegex(ValueError, "SHA256 mismatch"):
            REBUILD.prepare_rebuild(options)
        self.assertFalse((self.root / "rebuilds").exists())

    def test_unsafe_archives_are_rejected_before_any_write(self):
        names = ("../escape", "backend/../escape", "/backend/root", "other/entry",
                 "backend/evil\\name", "backend/.env.deploy", "backend/runtime/mysql/file",
                 "backend/" + "a" * 256, "backend/" + "/".join(["a"] * 65))
        for name in names:
            with self.subTest(name=name):
                entry = tarfile.TarInfo(name)
                entry.size = 1
                archive, digest = self.archive(extras=[(entry, b"x")])
                with self.assertRaises(ValueError):
                    REBUILD.prepare_rebuild(self.arguments(archive, digest))
                self.assertFalse((self.root / "rebuilds").exists())

    def test_archive_limits_reject_before_writing(self):
        options = self.arguments()
        with patch.object(REBUILD, "MAX_BYTES", 32):
            with self.assertRaisesRegex(ValueError, "size limit"):
                REBUILD.prepare_rebuild(options)
        with patch.object(REBUILD, "MAX_FILES", 2):
            with self.assertRaisesRegex(ValueError, "file count"):
                REBUILD.prepare_rebuild(options)
        oversized = tarfile.TarInfo("backend/oversized")
        oversized.size = REBUILD.MAX_BYTES + 1
        archive, digest = self.archive(extras=[(oversized, None)])
        with self.assertRaisesRegex(ValueError, "size limit"):
            REBUILD.prepare_rebuild(self.arguments(archive, digest))
        self.assertFalse((self.root / "rebuilds").exists())

    def test_sparse_files_and_directory_payloads_are_rejected(self):
        sparse = tarfile.TarInfo("backend/sparse")
        sparse.sparse = [(0, 1)]
        directory = tarfile.TarInfo("backend/directory")
        directory.type = tarfile.DIRTYPE
        directory.size = 1
        for entry in (sparse, directory):
            with self.subTest(name=entry.name), self.assertRaisesRegex(ValueError, "Invalid release entry"):
                REBUILD.members_checked([entry])

    def test_extended_metadata_reads_are_bounded_before_parser_allocation(self):
        entry = tarfile.TarInfo("backend/pax-header")
        entry.pax_headers = {"comment": "x" * 5000}
        archive, digest = self.archive(extras=[(entry, b"")])
        options = self.arguments(archive, digest)
        with patch.object(REBUILD, "MAX_METADATA_READ", 1024):
            with self.assertRaisesRegex(ValueError, "metadata read"):
                REBUILD.prepare_rebuild(options)
        with patch.object(REBUILD, "MAX_METADATA_BYTES", 1024):
            with self.assertRaisesRegex(ValueError, "metadata exceeds"):
                REBUILD.prepare_rebuild(options)
        self.assertFalse((self.root / "rebuilds").exists())
        with archive.open("rb") as source, REBUILD.ReleaseStream(source) as stream:
            with self.assertRaisesRegex(ValueError, "stream exceeds"):
                stream.seek(stream.max_offset() + 1)

    def test_extraction_stream_lengths_and_total_budget_are_enforced(self):
        destination = self.root / "extract"
        destination.mkdir()
        for name, content in (("short", b"short"), ("long", b"x" * 20)):
            entry = tarfile.TarInfo("backend/" + name)
            entry.size = 10
            archive = unittest.mock.Mock()
            archive.extractfile.return_value = io.BytesIO(content)
            with self.subTest(name=name), self.assertRaisesRegex(ValueError, "file length"):
                REBUILD.extract_checked(archive, [entry], destination)
            self.assertLessEqual((destination / entry.name).stat().st_size, entry.size)
        entry = tarfile.TarInfo("backend/over-budget")
        entry.size = 11
        with patch.object(REBUILD, "MAX_BYTES", 10):
            with self.assertRaisesRegex(ValueError, "size limit"):
                REBUILD.extract_checked(unittest.mock.Mock(), [entry], destination)
        self.assertFalse((destination / entry.name).exists())

    def test_links_and_special_files_are_rejected_before_writing(self):
        for kind in (tarfile.SYMTYPE, tarfile.LNKTYPE, tarfile.FIFOTYPE):
            with self.subTest(kind=kind):
                entry = tarfile.TarInfo("backend/link")
                entry.type = kind
                entry.linkname = "../../outside"
                archive, digest = self.archive(extras=[(entry, None)])
                with self.assertRaises(ValueError):
                    REBUILD.prepare_rebuild(self.arguments(archive, digest))
                self.assertFalse((self.root / "rebuilds").exists())

    def test_duplicates_and_missing_required_are_rejected(self):
        duplicate = tarfile.TarInfo(REBUILD.REQUIRED[0])
        duplicate.size = 1
        archive, digest = self.archive(extras=[(duplicate, b"x")])
        with self.assertRaisesRegex(ValueError, "Duplicate"):
            REBUILD.prepare_rebuild(self.arguments(archive, digest))
        archive, digest = self.archive(omitted=["frontend/dist/index.html"])
        with self.assertRaisesRegex(ValueError, "required"):
            REBUILD.prepare_rebuild(self.arguments(archive, digest))

    def test_missing_current_sql_is_rejected_before_preparing(self):
        archive, digest = self.archive(omitted=["backend/data/db/gozero-admin.sql"])
        with tarfile.open(archive) as release, self.assertRaisesRegex(ValueError, "required"):
            REBUILD.members_checked(release)
        self.assertFalse((self.root / "rebuilds").exists())

    def test_missing_migrations_are_rejected_before_preparing(self):
        archive, digest = self.archive(omitted=["backend/data/db/migrations/fixture.sql"])
        with tarfile.open(archive) as release, self.assertRaisesRegex(ValueError, "migration"):
            REBUILD.members_checked(release)
        self.assertFalse((self.root / "rebuilds").exists())

    def test_external_release_refused(self):
        with tempfile.TemporaryDirectory() as outside:
            archive = Path(outside).resolve() / "release.tar.gz"
            archive.write_bytes(b"not permitted")
            with self.assertRaisesRegex(ValueError, "inside"):
                REBUILD.prepare_rebuild(self.arguments(str(archive), "0" * 64))

    def test_old_compose_rejected_before_creating_environment(self):
        archive, digest = self.archive(contents={"backend/docker/demo-compose.yml": b"name: go-zero-admin-demo\nservices: {}\n"})
        with self.assertRaisesRegex(ValueError, "too old"):
            REBUILD.prepare_rebuild(self.arguments(archive, digest))
        self.assertFalse((self.root / "rebuilds").exists())

    def test_new_env_must_persist_isolated_project_and_root(self):
        deployment = self.root / "rebuilds/fixture"
        credentials = deployment / "runtime/.env.demo"
        credentials.parent.mkdir(parents=True)
        project = "go-zero-admin-demo-rebuild-fixture"
        credentials.write_text("DEPLOY_ROOT=" + str(deployment) + "\n", encoding="utf-8")
        # Test the resolved target directly; Windows does not require symlink privileges.
        with self.assertRaisesRegex(ValueError, "not persisted"):
            REBUILD.verify_new_environment(credentials, deployment, project)
        credentials.write_text("DEPLOY_ROOT=" + str(self.root) + "\nDEMO_PROJECT_NAME=" + project + "\n", encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "not persisted"):
            REBUILD.verify_new_environment(credentials, deployment, project)
        credentials.write_text("DEPLOY_ROOT=" + str(deployment) + "\nDEMO_PROJECT_NAME=" + project + "\n", encoding="utf-8")
        REBUILD.verify_new_environment(credentials, deployment, project)
        credentials.write_text("DEPLOY_ROOT=" + str(deployment) + "\nDEMO_PROJECT_NAME=" + project + "\nDEMO_PROJECT_NAME=go-zero-admin-demo\n", encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "not persisted"):
            REBUILD.verify_new_environment(credentials, deployment, project)

    def test_prepare_failure_preserves_old_data(self):
        data = self.root / "runtime/mysql/keep"
        data.parent.mkdir(parents=True)
        data.write_bytes(b"old")
        with patch.object(REBUILD.subprocess, "run", return_value=subprocess.CompletedProcess([], 1)):
            with self.assertRaisesRegex(RuntimeError, "Old deployment was untouched"):
                REBUILD.prepare_rebuild(self.arguments())
        self.assertEqual(data.read_bytes(), b"old")
        self.assertTrue((self.root / "rebuilds").is_dir())

    def test_optional_tls_copy_stays_separate(self):
        source = self.root / "runtime/caddy-data"
        source.mkdir(parents=True)
        (source / "fixture-cert").write_bytes(b"certificate fixture")
        destination = self.root / "rebuilds/fresh/runtime/caddy-data"
        REBUILD.copy_tls(source, destination, self.root)
        self.assertEqual((destination / "fixture-cert").read_bytes(), b"certificate fixture")
        self.assertEqual((source / "fixture-cert").read_bytes(), b"certificate fixture")


if __name__ == "__main__":
    unittest.main()
