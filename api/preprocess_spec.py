#!/usr/bin/env python3
"""Preprocess the upstream Trello OpenAPI spec to work around small bugs.

The upstream spec at https://dac-static.atlassian.com/cloud/trello/swagger.v3.json
has the following known issues that prevent oapi-codegen from generating code:

  1. PUT /cards/{idCard}/customFields declares no path parameter for ``{idCard}``.
  2. Several schemas use ``type: number`` together with ``format: integer`` (an
     invalid combination per the OpenAPI spec) or otherwise unknown numeric
     ``format`` values that kin-openapi rejects.
  3. A handful of operations share the same ``operationId``; oapi-codegen turns
     each operationId into a Go identifier so duplicates cause redeclaration
     errors at build time. We make every operationId unique by appending a
     numeric suffix to subsequent occurrences.
  4. Several path parameters use ``oneOf`` schemas (e.g. TrelloID + free-form
     string). oapi-codegen emits anonymous type aliases for these and ends up
     declaring duplicate identifiers across operations on the same path. Since
     path parameters are always strings on the wire we simplify their schemas
     to plain strings.

We patch the spec in-place (writing to the same file) so the generator can run.
This script is idempotent.
"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

# Formats kin-openapi accepts for numeric primitive types.
_NUMBER_FORMATS = {"float", "double"}
_INTEGER_FORMATS = {"int32", "int64"}


def _fix_numeric_formats(node: object) -> None:
    """Recursively normalise ``type``/``format`` pairs on numeric schemas."""
    if isinstance(node, dict):
        t = node.get("type")
        f = node.get("format")
        if t == "number" and f == "integer":
            node["type"] = "integer"
            node["format"] = "int64"
        elif t == "number" and f and f not in _NUMBER_FORMATS:
            node.pop("format", None)
        elif t == "integer" and f and f not in _INTEGER_FORMATS:
            node.pop("format", None)
        for v in node.values():
            _fix_numeric_formats(v)
    elif isinstance(node, list):
        for v in node:
            _fix_numeric_formats(v)


def patch(spec: dict) -> None:
    paths = spec.get("paths", {})

    # 1. Add missing path parameters that appear as ``{name}`` in the URL but
    #    aren't declared anywhere in the operation or path item.
    for url, item in paths.items():
        placeholders = set(re.findall(r"\{([^}]+)\}", url))
        path_level = {p["name"] for p in item.get("parameters", []) if p.get("in") == "path"}
        for verb, op in list(item.items()):
            if verb == "parameters" or not isinstance(op, dict):
                continue
            params = op.setdefault("parameters", [])
            declared = path_level | {p["name"] for p in params if p.get("in") == "path"}
            for missing in placeholders - declared:
                params.append({
                    "name": missing,
                    "in": "path",
                    "required": True,
                    "description": f"Auto-added missing path parameter {missing}",
                    "schema": {"type": "string"},
                })

    # 2. Walk every schema and fix invalid numeric type/format pairs.
    _fix_numeric_formats(spec)

    # 3. Make every operationId unique by appending ``-N`` to duplicates in
    #    sorted-path order so the generated Go identifiers don't collide.
    seen: dict[str, int] = {}
    for url in sorted(paths):
        item = paths[url]
        for verb, op in item.items():
            if verb == "parameters" or not isinstance(op, dict):
                continue
            op_id = op.get("operationId")
            if not op_id:
                continue
            count = seen.get(op_id, 0)
            if count:
                op["operationId"] = f"{op_id}-{count + 1}"
            seen[op_id] = count + 1

    # 4. Simplify all path parameter schemas to plain strings.
    for url, item in paths.items():
        def _simplify(params: list) -> None:
            for par in params:
                if par.get("in") == "path":
                    par["schema"] = {"type": "string"}

        _simplify(item.get("parameters", []))
        for verb, op in item.items():
            if verb == "parameters" or not isinstance(op, dict):
                continue
            _simplify(op.get("parameters", []))


def main(argv: list[str]) -> int:
    if len(argv) != 2:
        print("usage: preprocess_spec.py <spec.json>", file=sys.stderr)
        return 2
    path = Path(argv[1])
    spec = json.loads(path.read_text())
    patch(spec)
    path.write_text(json.dumps(spec, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
