"""Public-login migration checks using synthetic files and mocked SQL only."""
import copy
import importlib.util
import io
import os
import re
import subprocess
import tempfile
import unittest
from contextlib import ExitStack, redirect_stderr, redirect_stdout
from pathlib import Path
from unittest.mock import patch

MODULE_PATH = Path(__file__).resolve().parents[1] / "docker/demo-bootstrap.py"
SPEC = importlib.util.spec_from_file_location("demo_bootstrap", MODULE_PATH)
BOOTSTRAP = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(BOOTSTRAP)
REAL_SQL = BOOTSTRAP.sql

# Deliberately synthetic values: no application environment is read by this suite.
PRIVATE_PASSWORD = "fixture-private-password-not-live"
PRIVATE_HASH = "$2b$10$fixture-private-hash-not-live"
LEGACY_HASH = "$2b$10$fixture-visitor-hash-not-live"
PUBLIC_HASH = "$2b$10$fixture-public-hash-not-live"


class DemoBootstrapTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary.name).resolve()
        self.addCleanup(self.temporary.cleanup)
        self.backend = self.root / "backend"
        self.env_file = self.backend / "docker/.env.deploy"
        self.env_file.parent.mkdir(parents=True)
        template = self.backend / "bin/runtime/ai-rpc.yaml.template"
        template.parent.mkdir(parents=True)
        template.write_text("AI:\n  Enabled: false\n", encoding="utf-8")
        self.env_file.write_text(
            f"DEPLOY_ROOT={self.root}\nMYSQL_DATABASE=gozero-admin\n"
            "API_PORT=127.0.0.1:7001\nDEMO_PROJECT_NAME=go-zero-admin-demo\n"
            f"ADMIN_INITIAL_PASSWORD={PRIVATE_PASSWORD}\n"
            "API_KEY_DEEPSEEK=\nAPI_KEY_QWEN=\n", encoding="utf-8")
        self.credentials = self.root / "runtime/ADMIN_CREDENTIALS.txt"
        self.credentials.parent.mkdir(parents=True)
        self.write_credentials("admin", "demo / Demo@2026.")
        self.output = io.StringIO()
        self.queries = []
        self.hash_inputs = []
        self.marked = True
        self.role_active = True
        self.buttons = 0
        self.inheritance = 0
        self.scope = "self"
        self.grants = [f"{path}\tGET" for path in BOOTSTRAP.READ_PATHS]
        self.set_profile("legacy")
        self.mode_overrides = {self.env_file: 0o600, self.credentials: 0o600}
        original_stat = Path.stat

        def fixture_stat(path, *args, **kwargs):
            result = original_stat(path, *args, **kwargs)
            if path in self.mode_overrides:
                parts = list(result)
                parts[0] = (result.st_mode & ~0o777) | self.mode_overrides[path]
                return os.stat_result(parts)
            return result

        self.stack = ExitStack()
        self.addCleanup(self.stack.close)
        self.stack.enter_context(redirect_stdout(self.output))
        self.stack.enter_context(redirect_stderr(self.output))
        self.stack.enter_context(patch.object(BOOTSTRAP, "BACKEND", self.backend))
        self.stack.enter_context(patch.object(BOOTSTRAP, "ENV_FILE", self.env_file))
        self.stack.enter_context(patch.object(BOOTSTRAP.Path, "stat", fixture_stat))
        self.stack.enter_context(patch.object(BOOTSTRAP, "sql", self.mock_sql))
        self.stack.enter_context(patch.object(BOOTSTRAP, "password_hash", self.mock_hash))
        self.stack.enter_context(patch.object(
            BOOTSTRAP.subprocess, "run", side_effect=AssertionError("Real subprocess access is forbidden")))

    def write_credentials(self, username, visitor):
        self.credentials.write_text(
            f"Private administrator for this demo only.\nUsername: {username}\n"
            f"Password: {PRIVATE_PASSWORD}\nPublic visitors must use {visitor}\n", encoding="utf-8")
        self.credentials.chmod(0o600)

    def set_profile(self, profile):
        private_name, public_name = ("admin", "demo") if profile == "legacy" else ("demo-maintainer", "admin")
        self.users = {
            1: {"username": private_name, "authority_id": 1, "password": PRIVATE_HASH,
                "enable": 1, "session_version": 3},
            66: {"username": public_name, "authority_id": 9527, "password": LEGACY_HASH,
                 "enable": 1, "session_version": 4},
        }
        # Existing privileged associations may be preserved; only the public one is restricted.
        self.bindings = {1: [1, 102], 66: [9527]}

    def mock_hash(self, value):
        self.hash_inputs.append(value)
        self.assertEqual(value, BOOTSTRAP.PUBLIC_PASSWORD)
        return PUBLIC_HASH

    def mock_sql(self, statement):
        clean = " ".join(statement.split())
        self.queries.append(clean)
        if clean.startswith("START TRANSACTION;"):
            self.apply_login_transaction(clean)
            return ""
        self.assertTrue(clean.startswith("SELECT "), "Unexpected database write in public-login/check")
        if "information_schema.tables" in clean or "FROM sys_demo_bootstrap" in clean:
            return "1" if self.marked else "0"
        if "SELECT username,authority_id" in clean:
            return "\n".join(f"{row['username']}\t{row['authority_id']}"
                             for row in self.users.values() if row["enable"] == 1)
        if "SELECT ua.sys_authority_authority_id" in clean:
            username = re.search(r"u.username='([^']+)'", clean).group(1)
            return "\n".join(str(role) for identifier, row in self.users.items()
                             if row["username"] == username for role in self.bindings[identifier])
        if "SELECT COUNT(*) FROM sys_users WHERE username=" in clean:
            username = re.search(r"username='([^']+)'", clean).group(1)
            return str(sum(row["username"] == username for row in self.users.values()))
        if "FROM sys_authorities" in clean:
            return "1" if self.role_active else "0"
        if "FROM sys_authority_btns" in clean:
            return str(self.buttons)
        if "SELECT v1,v2 FROM casbin_rule" in clean:
            return "\n".join(reversed(self.grants))
        if "ptype='g'" in clean:
            return str(self.inheritance)
        if "SELECT scope FROM sys_role_data_scopes" in clean:
            return self.scope
        self.fail("Unexpected mock SQL query")

    def apply_login_transaction(self, statement):
        statements = [part.strip() for part in statement.split(";") if part.strip()]
        self.assertEqual(len(statements), 5)
        self.assertEqual(statements[0], "START TRANSACTION")
        self.assertEqual(statements[-1], "COMMIT")
        self.assertTrue(statements[3].startswith("INSERT INTO sys_audit_logs"))
        self.assertIn("'changePublicLogin'", statements[3])
        updates = copy.deepcopy(self.users)
        for query in statements[1:3]:
            matched = re.fullmatch(
                r"UPDATE sys_users SET (.*?) WHERE username='([^']+)' "
                r"AND authority_id=(\d+) AND enable=1 AND deleted_at IS NULL", query)
            self.assertIsNotNone(matched)
            assignments, username, role = matched.groups()
            rows = [row for row in updates.values()
                    if row["username"] == username and row["authority_id"] == int(role) and row["enable"] == 1]
            self.assertEqual(len(rows), 1)
            row = rows[0]
            expected_fields = {"username", "session_version", "updated_at"}
            if int(role) == 9527:
                expected_fields.add("password")
            fields = set()
            for assignment in assignments.split(","):
                field, value = assignment.strip().split("=", 1)
                fields.add(field)
                self.assertIn(field, expected_fields, "Migration must not expand permissions or change private password")
                if field in ("username", "password"):
                    self.assertTrue(value.startswith("'") and value.endswith("'"))
                    row[field] = value[1:-1]
                elif field == "session_version":
                    self.assertEqual(value, "session_version+1")
                    row[field] += 1
                else:
                    self.assertEqual(value, "NOW(3)")
            self.assertEqual(fields, expected_fields)
        self.users = updates

    def writes(self):
        return [query for query in self.queries if not query.startswith("SELECT ")]

    def test_legacy_migration_preserves_private_password_and_all_grants(self):
        users_before = copy.deepcopy(self.users)
        bindings_before = copy.deepcopy(self.bindings)
        grants_before = list(self.grants)
        original_open = os.open
        with patch.object(BOOTSTRAP.os, "open", wraps=original_open) as opened:
            BOOTSTRAP.public_login()
        self.assertEqual(self.users[1]["username"], "demo-maintainer")
        self.assertEqual(self.users[66]["username"], "admin")
        self.assertEqual(self.users[1]["password"], users_before[1]["password"])
        self.assertEqual(self.users[66]["password"], PUBLIC_HASH)
        self.assertEqual([self.users[1]["session_version"], self.users[66]["session_version"]], [4, 5])
        self.assertEqual([self.users[1]["authority_id"], self.users[66]["authority_id"]], [1, 9527])
        self.assertEqual(self.bindings, bindings_before)
        self.assertEqual(self.grants, grants_before)
        self.assertEqual(self.hash_inputs, ["123456"])
        self.assertEqual(len(self.writes()), 1)
        self.assertEqual(opened.call_count, 1)
        self.assertEqual(opened.call_args.args[2], 0o600)
        credential_text = self.credentials.read_text(encoding="utf-8")
        self.assertIn("Username: demo-maintainer\n", credential_text)
        self.assertIn("Password: " + PRIVATE_PASSWORD + "\n", credential_text)
        self.assertIn("admin / 123456 (read-only role 9527)", credential_text)
        self.assertNotIn(PRIVATE_PASSWORD, self.output.getvalue())
        self.assertNotIn(PRIVATE_HASH, self.output.getvalue())
        self.assertEqual(self.credentials.stat().st_mode & 0o777, 0o600)
        self.assertEqual(list(self.credentials.parent.glob(".ADMIN_CREDENTIALS-*.tmp")), [])

    def test_new_profile_reentry_does_not_reset_passwords_or_sessions(self):
        self.set_profile("current")
        self.write_credentials("demo-maintainer", "admin / 123456 (read-only role 9527).")
        users_before = copy.deepcopy(self.users)
        bytes_before = self.credentials.read_bytes()
        BOOTSTRAP.public_login()
        BOOTSTRAP.seed()
        self.assertEqual(self.users, users_before)
        self.assertEqual(self.credentials.read_bytes(), bytes_before)
        self.assertEqual(self.hash_inputs, [])
        self.assertEqual(self.writes(), [])

    def test_interrupted_credential_update_is_repaired_without_repeating_sql(self):
        self.set_profile("current")
        users_before = copy.deepcopy(self.users)
        BOOTSTRAP.public_login()
        self.assertEqual(self.users, users_before)
        self.assertEqual(self.hash_inputs, [])
        self.assertEqual(self.writes(), [])
        text = self.credentials.read_text(encoding="utf-8")
        self.assertIn("Username: demo-maintainer\n", text)
        self.assertIn(PRIVATE_PASSWORD, text)

    def test_public_admin_with_privileged_role_is_rejected(self):
        self.set_profile("current")
        self.users[66]["authority_id"] = 1
        with self.assertRaisesRegex(ValueError, "Unknown account layout"):
            BOOTSTRAP.public_login()
        self.assertEqual(self.writes(), [])
        self.assertEqual(self.hash_inputs, [])

    def test_unknown_or_extra_account_layout_is_rejected(self):
        self.users[1]["username"] = "unknown-maintainer"
        with self.assertRaisesRegex(ValueError, "Unknown account layout"):
            BOOTSTRAP.public_login()
        self.set_profile("legacy")
        self.users[99] = dict(self.users[66], username="another-visitor")
        with self.assertRaisesRegex(ValueError, "Unknown account layout"):
            BOOTSTRAP.public_login()
        self.assertEqual(self.writes(), [])

    def test_inactive_maintainer_name_collision_is_rejected(self):
        self.users[99] = dict(self.users[66], username="demo-maintainer", enable=2)
        with self.assertRaisesRegex(ValueError, "name already exists"):
            BOOTSTRAP.public_login()
        self.assertEqual(self.writes(), [])
        self.assertEqual(self.hash_inputs, [])

    def test_unmarked_database_is_not_migrated(self):
        self.marked = False
        with self.assertRaisesRegex(ValueError, "Initialize the dedicated demo"):
            BOOTSTRAP.public_login()
        self.assertEqual(self.writes(), [])

    def test_unsafe_credentials_and_unknown_legacy_file_are_rejected(self):
        self.mode_overrides[self.credentials] = 0o644
        with self.assertRaisesRegex(ValueError, "0600"):
            BOOTSTRAP.public_login()
        self.mode_overrides[self.credentials] = 0o600
        self.write_credentials("unknown-maintainer", "demo / Demo@2026.")
        with self.assertRaisesRegex(ValueError, "Unexpected private credential file"):
            BOOTSTRAP.public_login()
        self.assertEqual(self.writes(), [])
        self.assertEqual(self.hash_inputs, [])

    def test_unsafe_permissions_refuse_migration_before_hashing_or_writes(self):
        for field, unsafe in (("buttons", 1), ("inheritance", 1), ("scope", "all"), ("role_active", False)):
            with self.subTest(field=field):
                original = getattr(self, field)
                setattr(self, field, unsafe)
                with self.assertRaises(ValueError):
                    BOOTSTRAP.public_login()
                setattr(self, field, original)
        self.bindings[66].append(1)
        with self.assertRaisesRegex(ValueError, "bound only"):
            BOOTSTRAP.public_login()
        self.bindings[66] = [9527]
        self.grants.append("/v1/sys/audit/getAuditLogList\tGET")
        with self.assertRaisesRegex(ValueError, "exact read-only whitelist"):
            BOOTSTRAP.public_login()
        self.assertEqual(self.writes(), [])
        self.assertEqual(self.hash_inputs, [])

    def test_sql_failure_does_not_replace_private_credential_file(self):
        original_sql = self.mock_sql
        original_text = self.credentials.read_bytes()
        original_users = copy.deepcopy(self.users)

        def failing_sql(statement):
            if statement.startswith("START TRANSACTION;"):
                raise RuntimeError("Synthetic database failure")
            return original_sql(statement)

        with patch.object(BOOTSTRAP, "sql", failing_sql), self.assertRaises(RuntimeError):
            BOOTSTRAP.public_login()
        self.assertEqual(self.credentials.read_bytes(), original_text)
        self.assertEqual(self.users, original_users)

    def test_selected_database_is_preserved_for_demo_sql(self):
        self.env_file.write_text(self.env_file.read_text(encoding="utf-8").replace(
            "MYSQL_DATABASE=gozero-admin", "MYSQL_DATABASE=demo_fixture"), encoding="utf-8")
        self.assertEqual(BOOTSTRAP.read_env()["MYSQL_DATABASE"], "demo_fixture")
        with patch.object(BOOTSTRAP.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, "", "")) as run:
            REAL_SQL("SELECT DATABASE();")
        command = run.call_args.args[0]
        self.assertIn('"$MYSQL_DATABASE"', command[-1])
        self.assertEqual(run.call_args.kwargs["input"], "SELECT DATABASE();")

    def test_empty_database_imports_current_sql(self):
        snapshot = self.backend / "data/db/gozero-admin.sql"
        snapshot.parent.mkdir(parents=True)
        snapshot.write_text("CREATE TABLE fixture(id INT);\n", encoding="utf-8")
        statements = []

        def empty_sql(statement):
            statements.append(statement)
            return "0" if statement.startswith("SELECT ") else ""

        with patch.object(BOOTSTRAP, "sql", empty_sql):
            BOOTSTRAP.initialize_database()
        self.assertEqual(statements[-1], "CREATE TABLE fixture(id INT);\n")
        self.assertEqual(len([statement for statement in statements if not statement.startswith("SELECT ")]), 1)

    def test_existing_unmarked_database_is_never_imported(self):
        statements = []

        def existing_sql(statement):
            statements.append(statement)
            if "table_name='sys_demo_bootstrap'" in statement:
                return "0"
            return "3"

        with patch.object(BOOTSTRAP, "sql", existing_sql), self.assertRaisesRegex(ValueError, "non-empty"):
            BOOTSTRAP.initialize_database()
        self.assertTrue(all(statement.startswith("SELECT ") for statement in statements))

    def test_completed_demo_is_preserved_without_import(self):
        users_before = copy.deepcopy(self.users)
        BOOTSTRAP.initialize_database()
        self.assertEqual(self.users, users_before)
        self.assertEqual(self.writes(), [])

    def run_fresh_seed(self, dirty_table=None):
        migrations = self.backend / "data/db/migrations"
        migrations.mkdir(parents=True, exist_ok=True)
        (migrations / "fixture.sql").write_text("SELECT 1;\n", encoding="utf-8")
        statements = []

        def fresh_sql(statement):
            statements.append(statement)
            clean = " ".join(statement.split())
            if "information_schema.tables" in clean:
                return "0"
            if clean == "SELECT COUNT(*) FROM schema_migrations;":
                return "1"
            if clean == "SELECT COUNT(*) FROM sys_users;":
                return "1"
            if clean == "SELECT COUNT(*) FROM sys_users WHERE username='admin' AND authority_id=1 AND deleted_at IS NULL;":
                return "1"
            if clean.startswith("SELECT "):
                return "1" if dirty_table and f"FROM {dirty_table}" in clean else "0"
            return ""

        with patch.object(BOOTSTRAP, "sql", fresh_sql), patch.object(BOOTSTRAP, "check"), patch.object(
                BOOTSTRAP, "password_hash", side_effect=[PRIVATE_HASH, PUBLIC_HASH]) as hashed:
            if dirty_table:
                with self.assertRaisesRegex(ValueError, "existing business data"):
                    BOOTSTRAP.seed()
                self.assertEqual(hashed.call_count, 0)
                self.assertTrue(all(statement.startswith("SELECT ") for statement in statements))
            else:
                BOOTSTRAP.seed()
                self.assertEqual([call.args[0] for call in hashed.call_args_list], [PRIVATE_PASSWORD, "123456"])
                transaction = next(statement for statement in statements if statement.startswith("START TRANSACTION;"))
                self.assertIn("username='demo-maintainer'", transaction)
                self.assertIn("'admin','" + PUBLIC_HASH, transaction)
                self.assertIn("VALUES(@demo_user,9527)", transaction)
                self.assertIn("INSERT INTO sys_demo_bootstrap(id,completed_at)", transaction)

    def test_current_single_admin_seed_is_accepted(self):
        self.run_fresh_seed()

    def test_business_rows_refuse_seed_before_any_write(self):
        for table in ("sys_device_sessions", "sys_ai_runs", "sys_ai_conversations", "sys_ai_messages",
                      "sys_file_resources", "sys_file_references", "sys_departments", "sys_positions", "sys_audit_logs"):
            with self.subTest(table=table):
                self.run_fresh_seed(table)


if __name__ == "__main__":
    unittest.main()
