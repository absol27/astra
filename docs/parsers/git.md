```
principal:<author-email>
    --uses-->        resource:git                                          (VCS tool, one node across all commits)
    --carries_out--> step:commit:<slug>@<hash>                            (one step per commit)
    --consumes-->    artifact:gitcommit:<slug>@<parent-hash>              (parent commit(s))
    --consumes-->    artifact:gitfile:<slug>@<parent-hash>:<path>         (modified/deleted files before)
    --produces-->    artifact:gitcommit:<slug>@<hash>                     (this commit)
    --produces-->    artifact:gitfile:<slug>@<hash>:<path>                (added/modified files after)
```

## Field mapping

| Git concept                    | AStRA concept | ID format                                                    |
|--------------------------------|---------------|--------------------------------------------------------------|
| Commit                         | Step          | `step:commit:<host>/<owner>/<repo>@<hash>`                   |
| Author                         | Principal     | `principal:<email>`                                          |
| Git (VCS tool)                 | Resource      | `resource:git`                                               |
| Parent commit                  | ArtifactIn    | `artifact:gitcommit:<host>/<owner>/<repo>@<parent-hash>`     |
| Modified/deleted file (before) | ArtifactIn    | `artifact:gitfile:<slug>@<parent-hash>:<path>`               |
| This commit                    | ArtifactOut   | `artifact:gitcommit:<host>/<owner>/<repo>@<hash>`            |
| Added/modified file (after)    | ArtifactOut   | `artifact:gitfile:<slug>@<hash>:<path>`                      |

## Sample input

```sh
astra parse -f git -i https://github.com/pallets/flask.git -o parsed.json
```

## Expected parse output (`astra parse`)

One record of commit `91c6b3f` from `github.com/pallets/flask`:

```json
{
  "source": "go-git",
  "normalized_at": 1774360050,
  "mapped": [
    {
      "step": {
        "id": "step:commit:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d",
        "label": "Commit",
        "kind": "step",
        "attrs": {
          "phase": "source",
          "message": "remove unicode host test"
        }
      },
      "principal": {
        "id": "principal:davidism@gmail.com",
        "label": "David Lord",
        "kind": "principal",
        "attrs": {
          "email": "davidism@gmail.com"
        }
      },
      "artifacts_in": [
        {
          "id": "artifact:gitcommit:github.com/pallets/flask@4cae5d8e411b1e69949d8fae669afeacbd3e5908",
          "label": "4cae5d8e411b1e69949d8fae669afeacbd3e5908",
          "kind": "git-commit",
          "attrs": { "role": "parent", "index": "0" }
        },
        {
          "id": "artifact:gitfile:github.com/pallets/flask@4cae5d8e411b1e69949d8fae669afeacbd3e5908:tests/test_reqctx.py",
          "label": "tests/test_reqctx.py",
          "kind": "git-file",
          "attrs": { "hash": "78561f520d5ab28fe83f5c24bbc9c269cbe41874", "size": "8698", "mode": "0100644" }
        }
      ],
      "artifacts_out": [
        {
          "id": "artifact:gitcommit:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d",
          "label": "91c6b3fecf36b1f04554e57cc4060ccb737a445d",
          "kind": "git-commit",
          "attrs": {
            "author": "davidism@gmail.com",
            "message": "remove unicode host test",
            "time": "1774360050"
          }
        },
        {
          "id": "artifact:gitfile:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d:tests/test_reqctx.py",
          "label": "tests/test_reqctx.py",
          "kind": "git-file",
          "attrs": { "content-hash": "3c5d5332adafaee65c7a79643b7f65cd8f366095", "size": "8197", "mode": "0100644" }
        }
      ],
      "resources": [
        {
          "id": "resource:git",
          "label": "git",
          "kind": "vcs",
          "attrs": null
        }
      ]
    }
    // ... one record per commit
  ]
}
```

## Expected map output (`astra map`)

```json
{
  "artifacts": [
    {
      "id": "artifact:gitcommit:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d",
      "kind": "git-commit",
      "name": "91c6b3fecf36b1f04554e57cc4060ccb737a445d",
      "metadata": { "message": "remove unicode host test", "author": "davidism@gmail.com", "time": "1774360050" }
    }
    // ... commit and file artifacts for all commits
  ],
  "steps": [
    {
      "id": "step:commit:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d",
      "command": "",
      "timestamp": "",
      "architecture": "",
      "environment": null,
      "metadata": { "phase": "source", "message": "remove unicode host test" }
    }
    // ... 5,524 steps total
  ],
  "principals": [
    {
      "id": "principal:davidism@gmail.com",
      "name": "David Lord",
      "trust_level": "unknown",
      "builder": "",
      "metadata": { "email": "davidism@gmail.com" }
    }
    // ... 872 principals total
  ],
  "resources": [
    {
      "id": "resource:git",
      "type": "vcs",
      "uri": "",
      "format": "git"
    }
  ],
  "edges": [
    { "source": "principal:davidism@gmail.com",                                                   "target": "resource:git",                                                                                     "relation": "uses"        },
    { "source": "resource:git",                                                                   "target": "step:commit:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d",                  "relation": "carries_out" },
    { "source": "step:commit:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d", "target": "artifact:gitcommit:github.com/pallets/flask@4cae5d8e411b1e69949d8fae669afeacbd3e5908",            "relation": "consumes"    },
    { "source": "step:commit:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d", "target": "artifact:gitfile:github.com/pallets/flask@4cae5d8e...:tests/test_reqctx.py",                      "relation": "consumes"    },
    { "source": "step:commit:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d", "target": "artifact:gitcommit:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d",            "relation": "produces"    },
    { "source": "step:commit:github.com/pallets/flask@91c6b3fecf36b1f04554e57cc4060ccb737a445d", "target": "artifact:gitfile:github.com/pallets/flask@91c6b3f...:tests/test_reqctx.py",                       "relation": "produces"    }
    // ... repeated for all 5,524 commits
  ]
}
```

Edge counts for flask: 872 `uses`, 5,524 `carries_out`, 23,195 `consumes`, 21,910 `produces` = **51,501 total**.
