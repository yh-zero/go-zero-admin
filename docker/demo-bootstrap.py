#!/usr/bin/env python3
"""Prepare a dedicated public demo. Refuse to reseed an existing business DB."""
import ipaddress
import os
import re
import secrets
import subprocess
import sys
import uuid
import warnings
from pathlib import Path

BACKEND = Path(__file__).resolve().parent.parent
ENV_FILE = BACKEND / "docker/.env.deploy"
PUBLIC_USERNAME = "admin"
PRIVATE_USERNAME = "demo-maintainer"
PUBLIC_PASSWORD = "123456"
# The public login uses role 9527 only. Privileged role 1 keeps a private password.
READ_PATHS = [
    "/v1/sys/menu/getMenu", "/v1/sys/menu/getMenuList",
    "/v1/sys/menu/getBaseMenuById", "/v1/sys/menu/getBaseMenuTree",
    "/v1/sys/api/getApiList", "/v1/sys/api/getAllApiList",
    "/v1/sys/dictionary/getSysDictionaryList",
    "/v1/sys/dictionary/getSysDictionaryDetails",
    "/v1/sys/dictionary/getSysDictionaryInfoList",
    "/v1/sys/dictionary/getSysDictionaryInfoListDetailsById",
    "/v1/sys/organization/departments", "/v1/sys/organization/positions",
    "/v1/sys/files/list", "/v1/ai/info", "/v1/ai/conversations",
    "/v1/ai/conversations/:id/messages", "/v1/ai/runs/:id",
]


def inside(path, root):
    resolved = path.resolve()
    if not resolved.is_relative_to(root):
        raise ValueError("Project path escapes the selected deployment directory")
    return resolved


def private_write(path, content):
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(descriptor, "w", encoding="utf-8", newline="\n") as target:
        target.write(content)


def prepare(directory, domain, address):
    root = Path(directory)
    if not root.is_absolute() or not root.is_dir():
        raise ValueError("Select an existing absolute project directory")
    root = root.resolve()
    inside(BACKEND, root)
    if not re.fullmatch(r"[a-zA-Z0-9](?:[a-zA-Z0-9.-]*[a-zA-Z0-9])?", domain) or "." not in domain:
        raise ValueError("Invalid demo domain")
    ipaddress.IPv4Address(address)
    project = os.environ.get("DEMO_PROJECT_NAME", "go-zero-admin-demo")
    if not re.fullmatch(r"go-zero-admin-demo(?:-rebuild-[a-z0-9-]+)?", project):
        raise ValueError("Invalid demo Compose project name")
    runtime = inside(root / "runtime", root)
    runtime.mkdir(mode=0o700, exist_ok=True)
    for name in ("mysql", "redis", "etcd", "caddy-data", "caddy-config", "backups"):
        inside(runtime / name, root).mkdir(mode=0o700, exist_ok=True)
    credentials = inside(runtime / ".env.demo", root)
    if not credentials.exists():
        # Generate secrets on the server; no original local secrets are uploaded.
        values = {
            "DEPLOY_ROOT": str(root), "DEMO_DOMAIN": domain, "DEMO_IP": address,
            "DEMO_PROJECT_NAME": project,
            "MYSQL_ROOT_PASSWORD": secrets.token_hex(32), "MYSQL_DATABASE": "gozero-admin",
            "REDIS_PASSWORD": secrets.token_hex(32), "JWT_ACCESS_SECRET": secrets.token_hex(32),
            "JWT_ACCESS_EXPIRE": "1800", "RPC_AUTH_APP": "applet-api",
            "RPC_AUTH_TOKEN": secrets.token_hex(32), "DEFAULT_USER_PASSWORD": secrets.token_urlsafe(32),
            "ADMIN_INITIAL_PASSWORD": secrets.token_urlsafe(32), "API_PORT": "127.0.0.1:7001",
            "API_KEY_DEEPSEEK": "", "API_KEY_QWEN": "",
        }
        private_write(credentials, "".join(f"{key}={value}\n" for key, value in values.items()))
        private_write(runtime / "ADMIN_CREDENTIALS.txt", "Private administrator for this demo only.\nUsername: demo-maintainer\nPassword: " + values["ADMIN_INITIAL_PASSWORD"] + "\nPublic visitors must use admin / 123456 (read-only role 9527).\n")
    if ENV_FILE.is_symlink():
        if ENV_FILE.resolve() != credentials:
            raise ValueError("Existing environment link targets another deployment")
    elif ENV_FILE.exists():
        raise ValueError("Preserve the existing deployment env; automatic replacement refused")
    else:
        ENV_FILE.symlink_to(credentials)
    read_env()  # Validate reused settings, too.
    print("Demo settings prepared; secrets remain in the private runtime directory")


def read_env():
    values = {}
    for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
        if line and not line.startswith("#"):
            key, value = line.split("=", 1)
            values[key] = value
    root = Path(values["DEPLOY_ROOT"]).resolve()
    inside(BACKEND, root)
    inside(ENV_FILE, root)
    if ENV_FILE.stat().st_mode & 0o077:
        raise ValueError("Deployment environment must have private 0600 permissions")
    if not re.fullmatch(r"[A-Za-z0-9_-]+", values.get("MYSQL_DATABASE", "")) or values.get("API_PORT") != "127.0.0.1:7001":
        raise ValueError("Unexpected demo database or API binding")
    if values.get("API_KEY_DEEPSEEK") or values.get("API_KEY_QWEN"):
        raise ValueError("This public demo must not contain a model key")
    if not re.fullmatch(r"go-zero-admin-demo(?:-rebuild-[a-z0-9-]+)?", values.get("DEMO_PROJECT_NAME", "go-zero-admin-demo")):
        raise ValueError("Invalid demo Compose project name")
    template = (BACKEND / "bin/runtime/ai-rpc.yaml.template").read_text(encoding="utf-8")
    if not re.search(r"(?m)^\s*Enabled:\s*false\s*$", template):
        raise ValueError("Public demo AI must remain disabled")
    return values


def sql(statement):
    command = ["docker", "compose", "--env-file", str(ENV_FILE), "-f", str(BACKEND / "docker/demo-compose.yml"),
               "exec", "-T", "mysql", "sh", "-ec",
               'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql --default-character-set=utf8mb4 --batch --skip-column-names "$MYSQL_DATABASE"']
    environment = os.environ.copy()
    for selector in ("COMPOSE_PROJECT_NAME", "DEMO_PROJECT_NAME", "DEPLOY_ROOT"):
        environment.pop(selector, None)
    result = subprocess.run(command, input=statement, text=True, encoding="utf-8", capture_output=True, check=False, env=environment)
    if result.returncode:
        # Never print raw SQL/errors; they may include private password hashes.
        raise RuntimeError("Demo database operation failed; no success marker was written")
    return result.stdout.strip()


def completed():
    if sql("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='sys_demo_bootstrap';") == "0":
        return False
    return sql("SELECT COUNT(*) FROM sys_demo_bootstrap WHERE id=1;") == "1"


def initialize_database():
    """Import the current SQL only into a completely empty demo database."""
    read_env()
    if sql("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE();") != "0":
        if not completed():
            raise ValueError("Demo database is non-empty and has no completed bootstrap; existing data will not be altered")
        print("Existing initialized demo database preserved; SQL import skipped")
        return
    snapshot = BACKEND / "data/db/gozero-admin.sql"
    sql(snapshot.read_text(encoding="utf-8-sig"))
    print("Current SQL imported into the selected empty demo database")


def check(public_username=PUBLIC_USERNAME, private_username=PRIVATE_USERNAME):
    read_env()
    if not completed():
        raise ValueError("Run demo init first; public start requires a completed bootstrap")
    accounts = sql("SELECT username,authority_id FROM sys_users WHERE enable=1 AND deleted_at IS NULL;")
    if sorted(accounts.splitlines()) != sorted([f"{private_username}\t1", f"{public_username}\t9527"]):
        raise ValueError("Public login must have read-only role 9527; role 1 is private")
    bindings = sql(f"""SELECT ua.sys_authority_authority_id FROM sys_user_authority ua
        JOIN sys_users u ON u.id=ua.sys_user_id
        WHERE u.username='{public_username}' AND u.deleted_at IS NULL;""")
    if bindings != "9527":
        raise ValueError("Demo must be bound only to role 9527")
    if sql("SELECT COUNT(*) FROM sys_authorities WHERE authority_id=9527 AND deleted_at IS NULL;") != "1":
        raise ValueError("The public demo role must exist and remain active")
    if sql("SELECT COUNT(*) FROM sys_authority_btns WHERE authority_id=9527;") != "0":
        raise ValueError("The public demo role must not contain button grants")
    grants = sql("SELECT v1,v2 FROM casbin_rule WHERE ptype='p' AND v0='9527';")
    if sorted(grants.splitlines()) != sorted(f"{path}\tGET" for path in READ_PATHS):
        raise ValueError("Demo permissions must match the exact read-only whitelist")
    if sql("SELECT COUNT(*) FROM casbin_rule WHERE ptype='g' AND (v0='9527' OR v1='9527');") != "0":
        raise ValueError("The public demo role must not participate in role inheritance")
    if sql("SELECT scope FROM sys_role_data_scopes WHERE authority_id=9527;") != "self":
        raise ValueError("The public demo data scope must remain self")
    print("Demo bootstrap, read-only grants and disabled AI checked")


def password_hash(value):
    with warnings.catch_warnings():
        warnings.simplefilter("ignore", DeprecationWarning)
        import crypt
    if not hasattr(crypt, "METHOD_BLOWFISH"):
        raise RuntimeError("bcrypt is unavailable; do not seed using a weaker hash")
    result = crypt.crypt(value, crypt.mksalt(crypt.METHOD_BLOWFISH, rounds=1024))
    if not result.startswith(("$2a$", "$2b$", "$2y$")):
        raise RuntimeError("Safe password hashing failed")
    return result


def public_login():
    """Rename the existing read-only visitor; never grant public role 1."""
    values = read_env()
    if not completed():
        raise ValueError("Initialize the dedicated demo before changing its public login")
    accounts = sql("SELECT username,authority_id FROM sys_users WHERE enable=1 AND deleted_at IS NULL;")
    current = sorted([f"{PRIVATE_USERNAME}\t1", f"{PUBLIC_USERNAME}\t9527"])
    legacy = sorted(["admin\t1", "demo\t9527"])
    if sorted(accounts.splitlines()) not in (current, legacy):
        raise ValueError("Unknown account layout; public login update refused")
    root = Path(values["DEPLOY_ROOT"]).resolve()
    credential_file = inside(root / "runtime/ADMIN_CREDENTIALS.txt", root)
    if credential_file.stat().st_mode & 0o077:
        raise ValueError("Private administrator credentials must remain 0600")
    credential_text = credential_file.read_text(encoding="utf-8")
    if sorted(accounts.splitlines()) == legacy:
        check("demo", "admin")
        if sql(f"SELECT COUNT(*) FROM sys_users WHERE username='{PRIVATE_USERNAME}';") != "0":
            raise ValueError("Private administrator name already exists; update refused")
        if "Username: admin\n" not in credential_text:
            raise ValueError("Unexpected private credential file; update refused")
        visitor_hash = password_hash(PUBLIC_PASSWORD)
        sql(f"""START TRANSACTION;
UPDATE sys_users SET username='{PRIVATE_USERNAME}',session_version=session_version+1,updated_at=NOW(3)
 WHERE username='admin' AND authority_id=1 AND enable=1 AND deleted_at IS NULL;
UPDATE sys_users SET username='{PUBLIC_USERNAME}',password='{visitor_hash}',session_version=session_version+1,updated_at=NOW(3)
 WHERE username='demo' AND authority_id=9527 AND enable=1 AND deleted_at IS NULL;
INSERT INTO sys_audit_logs(event_type,module,action,result,status_code,params)
 VALUES('operation','deployment','changePublicLogin','success',200,'');
COMMIT;
""")
    check()
    updated = credential_text.replace("Username: admin\n", f"Username: {PRIVATE_USERNAME}\n")
    updated = updated.replace("Public visitors must use demo / Demo@2026.", "Public visitors must use admin / 123456 (read-only role 9527).")
    if updated != credential_text:
        temporary = credential_file.with_name(f".ADMIN_CREDENTIALS-{uuid.uuid4().hex}.tmp")
        private_write(temporary, updated)
        temporary.replace(credential_file)
    print("Public admin login configured with read-only role 9527; private role 1 preserved")


def seed():
    values = read_env()
    if completed():
        public_login()
        print("Demo already initialized; credentials and existing rows preserved")
        return
    expected = len(list((BACKEND / "data/db/migrations").glob("*.sql")))
    if sql("SELECT COUNT(*) FROM schema_migrations;") != str(expected):
        raise ValueError("Apply the complete migration chain before demo bootstrap")
    checks = [
        ("SELECT COUNT(*) FROM sys_users;", "1"),
        ("SELECT COUNT(*) FROM sys_users WHERE username='admin' AND authority_id=1 AND deleted_at IS NULL;", "1"),
        ("SELECT COUNT(*) FROM sys_users WHERE username='demo';", "0"),
        ("SELECT COUNT(*) FROM sys_authorities WHERE authority_id=9527;", "0"),
        ("SELECT COUNT(*) FROM sys_device_sessions;", "0"),
        ("SELECT COUNT(*) FROM sys_ai_runs;", "0"),
        ("SELECT COUNT(*) FROM sys_ai_conversations;", "0"),
        ("SELECT COUNT(*) FROM sys_ai_messages;", "0"),
        ("SELECT COUNT(*) FROM sys_file_resources;", "0"),
        ("SELECT COUNT(*) FROM sys_file_references;", "0"),
        ("SELECT COUNT(*) FROM sys_departments;", "0"),
        ("SELECT COUNT(*) FROM sys_positions;", "0"),
        ("SELECT COUNT(*) FROM sys_audit_logs;", "0"),
    ]
    for query, expected_value in checks:
        if sql(query) != expected_value:
            raise ValueError("Fresh demo database assertion failed; existing business data will not be altered")
    # Ubuntu 24.04/Python 3.12 provides libcrypt Blowfish. No weak hash fallback.
    admin_hash = password_hash(values["ADMIN_INITIAL_PASSWORD"])
    demo_hash = password_hash(PUBLIC_PASSWORD)
    grants = ",\n".join(f"('p','9527','{path}','GET')" for path in READ_PATHS)
    sql("CREATE TABLE IF NOT EXISTS sys_demo_bootstrap(id TINYINT PRIMARY KEY,completed_at DATETIME(3) NOT NULL);")
    sql(f"""START TRANSACTION;
UPDATE sys_users SET enable=2,session_version=session_version+1,phone='',email='',header_img='';
UPDATE sys_users SET username='{PRIVATE_USERNAME}',enable=1,password='{admin_hash}' WHERE username='admin' AND authority_id=1 AND deleted_at IS NULL;
INSERT INTO sys_authorities(created_at,updated_at,authority_id,authority_name,parent_id,default_router)
VALUES(NOW(3),NOW(3),9527,'公开只读演示',0,'index');
INSERT INTO sys_users(created_at,updated_at,uuid,username,password,nick_name,side_mode,header_img,base_color,active_color,authority_id,phone,email,enable,session_version)
VALUES(NOW(3),NOW(3),'{uuid.uuid4()}','{PUBLIC_USERNAME}','{demo_hash}','公开演示','dark','','#fff','#1890ff',9527,'','',1,1);
SET @demo_user=LAST_INSERT_ID();
INSERT INTO sys_user_authority(sys_user_id,sys_authority_authority_id) VALUES(@demo_user,9527);
INSERT INTO sys_role_data_scopes(authority_id,scope) VALUES(9527,'self');
INSERT INTO sys_authority_menus(sys_base_menu_id,sys_authority_authority_id)
WITH RECURSIVE permitted AS (
 SELECT id,parent_id FROM sys_base_menus WHERE deleted_at IS NULL AND name IN
 ('index','menu','api','dictionary','files','organization-departments','organization-positions','ai-agent')
 UNION DISTINCT
 SELECT m.id,m.parent_id FROM sys_base_menus m JOIN permitted p ON m.id=p.parent_id WHERE m.deleted_at IS NULL
) SELECT DISTINCT id,9527 FROM permitted;
INSERT INTO casbin_rule(ptype,v0,v1,v2) VALUES {grants};
INSERT INTO sys_departments(parent_id,name,code,sort,status,leader) VALUES(0,'示例部门','DEMO-ROOT',1,1,'');
INSERT INTO sys_positions(name,code,sort,status) VALUES('示例岗位','DEMO-POSITION',1,1);
UPDATE sys_policy_versions SET version=version+1,updated_at=NOW(3) WHERE id=1;
INSERT INTO sys_audit_logs(event_type,module,action,result,status_code,params)
VALUES('operation','deployment','bootstrapDemo','success',200,'');
INSERT INTO sys_demo_bootstrap(id,completed_at) VALUES(1,NOW(3));
COMMIT;
""")
    check()
    print("Fresh public demo initialized; private administrator and read-only visitor account ready")


if __name__ == "__main__":
    try:
        if len(sys.argv) == 5 and sys.argv[1] == "prepare":
            prepare(*sys.argv[2:])
        elif len(sys.argv) == 2 and sys.argv[1] == "init-db":
            initialize_database()
        elif len(sys.argv) == 2 and sys.argv[1] == "seed":
            seed()
        elif len(sys.argv) == 2 and sys.argv[1] == "check":
            check()
        elif len(sys.argv) == 2 and sys.argv[1] == "public-login":
            public_login()
        else:
            raise ValueError("Use prepare ROOT DOMAIN IP, init-db, seed, check or public-login")
    except (ValueError, RuntimeError, OSError, KeyError) as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
