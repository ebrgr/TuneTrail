#!/usr/bin/env python3
"""Build an experimental v8 binary without staging a second copy of listens."""

import argparse
from pathlib import Path
import subprocess
import tarfile
import tempfile


def without_staging(sql):
    """Move both listen assertions ahead of the swap while both tables exist."""
    start = sql.index("INSERT INTO identity_assertion VALUES ('listens per profile'")
    end = sql.index("INSERT INTO identity_assertion VALUES ('foreign keys'", start)
    checks = sql[start:end].replace("FROM listens", "FROM listens_new").replace("FROM old_listens", "FROM listens")
    columns = "rowid,profile_id,track_id,listened_at,user_id,client"
    checks = checks.replace("SELECT * FROM listens", f"SELECT {columns} FROM listens")
    sql = sql[:start] + sql[end:]
    sql = sql.replace("CREATE TEMP TABLE old_listens AS SELECT rowid AS legacy_rowid, * FROM listens;\n", "")
    sql = sql.replace(
        "SELECT legacy_rowid,profile_id,track_id,listened_at,user_id,client,'legacy' FROM old_listens;",
        f"SELECT {columns},'legacy' FROM listens;",
    )
    return sql.replace("DROP TABLE listens;", checks + "DROP TABLE listens;").replace("DROP TABLE old_listens;\n", "")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path)
    parser.add_argument("--ref", default="HEAD")
    args = parser.parse_args()
    output = args.output.resolve()
    with tempfile.TemporaryDirectory(prefix="tt133-no-temp-", dir="/tmp") as tmp:
        root = Path(tmp)
        archive = root / "source.tar"
        with archive.open("wb") as target:
            subprocess.run(["git", "archive", args.ref], stdout=target, check=True)
        with tarfile.open(archive) as source:
            source.extractall(root, filter="data")
        migration = root / "db/migrations_sqlite/008_recording_identity.sql"
        migration.write_text(without_staging(migration.read_text()))
        subprocess.run(["go", "build", "-o", str(output), "./cmd/api"], cwd=root, check=True)


if __name__ == "__main__":
    main()
