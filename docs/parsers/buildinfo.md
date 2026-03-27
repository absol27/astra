```
principal:Debian
    --uses-->        pkg:deb/debian/<source>@<upstream>?arch=source   (resource: source tarball)
    --uses-->        pkg:deb/debian/<dep>@<ver>                        (resource: each build dependency)
    --carries_out--> step:build:deb/<source>@<version>                (one step per buildinfo file)
    --produces-->    pkg:deb/debian/<pkg>@<ver>?arch=<arch>            (×M output .deb files)
```

## Field mapping

| `.buildinfo` field         | AStRA concept    | Notes                                              |
|----------------------------|------------------|----------------------------------------------------|
| `Source` + `Version`       | Step ID          | `step:build:deb/<source>@<version>`                |
| `Build-Date`               | Step timestamp   | `attrs["timestamp"]`                               |
| `Build-Architecture`       | Step attrs       | `attrs["architecture"]`                            |
| `Build-Origin`             | Principal ID     | `principal:Debian`       |
| PGP signature key ID       | Principal attrs  | `attrs["pgp_key_id"]`                              |
| Source tarball (inferred)  | Resource         | `pkg:deb/debian/<source>@<upstream-version>`       |
| `Installed-Build-Depends`  | Resources        | Each pinned dep as a purl                          |
| `Checksums-Sha256` `.deb`  | ArtifactsOut     | Each output `.deb` produced by step                |

Identifiers use [Package URL (purl)](https://github.com/package-url/purl-spec) format:
`pkg:deb/debian/<name>@<version>`

## Sample input

```
Source: haskell-hinotify
Version: 0.4-2
Build-Origin: Debian
Build-Architecture: arm64
Build-Date: Sun, 07 Jun 2020 01:58:10 +0000
Checksums-Sha256:
 0babd2e40ddd18827e92a339525730c65eb5b691a668f5921bee0bbfe56a85fd 98920 libghc-hinotify-dev_0.4-2_arm64.deb
 3c09481578956fdb6e497573a04f7eea54b7b39c0ec80b5568b057009f509854 83872 libghc-hinotify-prof_0.4-2_arm64.deb
Installed-Build-Depends:
 autoconf (= 2.69-11.1),
 automake (= 1:1.16.2-1),
 ... (189 deps total)
```

## Expected parse output (`astra parse`)

```json
{
  "source": "buildinfo",
  "normalized_at": 1234567890,
  "mapped": [
    {
      "step": {
        "id": "step:build:deb/haskell-hinotify@0.4-2",
        "label": "dpkg-buildpackage",
        "kind": "build",
        "attrs": {
          "architecture": "arm64",
          "command": "dpkg-buildpackage",
          "timestamp": "Sun, 07 Jun 2020 01:58:10 +0000"
        }
      },
      "principal": {
        "id": "principal:Debian",
        "label": "Debian Build Infrastructure",
        "kind": "principal",
        "attrs": {
          "pgp_key_id": "9bb8aaf879f91bf7"
        }
      },
      "artifacts_in": [],
      "artifacts_out": [
        {
          "id": "pkg:deb/debian/libghc-hinotify-dev@0.4-2",
          "label": "libghc-hinotify-dev_0.4-2_arm64.deb",
          "kind": "deb",
          "attrs": {
            "filename": "libghc-hinotify-dev_0.4-2_arm64.deb",
            "hash": "0babd2e40ddd18827e92a339525730c65eb5b691a668f5921bee0bbfe56a85fd",
            "purl": "pkg:deb/debian/libghc-hinotify-dev@0.4-2",
            "size": "98920",
            "version": "0.4-2"
          }
        },
        {
          "id": "pkg:deb/debian/libghc-hinotify-prof@0.4-2",
          "label": "libghc-hinotify-prof_0.4-2_arm64.deb",
          "kind": "deb",
          "attrs": {
            "filename": "libghc-hinotify-prof_0.4-2_arm64.deb",
            "hash": "3c09481578956fdb6e497573a04f7eea54b7b39c0ec80b5568b057009f509854",
            "purl": "pkg:deb/debian/libghc-hinotify-prof@0.4-2",
            "size": "83872",
            "version": "0.4-2"
          }
        }
      ],
      "resources": [
        {
          "id": "pkg:deb/debian/haskell-hinotify@0.4?arch=source",
          "label": "haskell-hinotify_0.4.orig.tar.xz",
          "kind": "tarball",
          "attrs": {
            "format": "orig.tar.xz",
            "purl": "pkg:deb/debian/haskell-hinotify@0.4?arch=source"
          }
        },
        {
          "id": "pkg:deb/debian/autoconf@2.69-11.1",
          "label": "autoconf",
          "kind": "deb",
          "attrs": {
            "purl": "pkg:deb/debian/autoconf@2.69-11.1",
            "version": "2.69-11.1"
          }
        },
        {
          "id": "pkg:deb/debian/automake@1:1.16.2-1",
          "label": "automake",
          "kind": "deb",
          "attrs": {
            "purl": "pkg:deb/debian/automake@1:1.16.2-1",
            "version": "1:1.16.2-1"
          }
        }
        // ... 187 more build dependencies
      ]
    }
  ]
}
```

## Expected map output (`astra map`)

```json
{
  "artifacts": [
    {
      "id": "pkg:deb/debian/libghc-hinotify-dev@0.4-2?arch=arm64",
      "kind": "deb",
      "name": "libghc-hinotify-dev_0.4-2_arm64.deb",
      "version": "0.4-2",
      "metadata": { ... }
    },
    {
      "id": "pkg:deb/debian/libghc-hinotify-prof@0.4-2?arch=arm64",
      "kind": "deb",
      "name": "libghc-hinotify-prof_0.4-2_arm64.deb",
      "version": "0.4-2",
      "metadata": { ... }
    }
  ],
  "steps": [
    {
      "id": "step:build:deb/haskell-hinotify@0.4-2",
      "command": "dpkg-buildpackage",
      "timestamp": "Sun, 07 Jun 2020 01:58:10 +0000",
      "architecture": "arm64",
      "environment": null,
      "metadata": {
        "architecture": "arm64",
        "command": "dpkg-buildpackage",
        "timestamp": "Sun, 07 Jun 2020 01:58:10 +0000"
      }
    }
  ],
  "principals": [
    {
      "id": "principal:Debian",
      "name": "Debian Build Infrastructure",
      "trust_level": "unknown",
      "builder": "",
      "metadata": { "pgp_key_id": "9bb8aaf879f91bf7" }
    }
  ],
  "resources": [
    {
      "id": "pkg:deb/debian/haskell-hinotify@0.4?arch=source",
      "type": "tarball",
      "uri": "",
      "format": "orig.tar.xz"
    },
    // ... 189 build dependency resources
  ],
  "edges": [
    { "source": "principal:Debian",                       "target": "pkg:deb/debian/haskell-hinotify@0.4?arch=source",  "relation": "uses"        },
    { "source": "principal:Debian",                       "target": "pkg:deb/debian/autoconf@2.69-11.1",                "relation": "uses"        },
    // ... 188 more uses edges (one per build dependency)
    { "source": "pkg:deb/debian/haskell-hinotify@0.4?arch=source", "target": "step:build:deb/haskell-hinotify@0.4-2",  "relation": "carries_out" },
    { "source": "pkg:deb/debian/autoconf@2.69-11.1",      "target": "step:build:deb/haskell-hinotify@0.4-2",           "relation": "carries_out" },
    // ... 188 more carries_out edges (one per build dependency)
    { "source": "step:build:deb/haskell-hinotify@0.4-2",  "target": "pkg:deb/debian/libghc-hinotify-dev@0.4-2?arch=arm64",  "relation": "produces" },
    { "source": "step:build:deb/haskell-hinotify@0.4-2",  "target": "pkg:deb/debian/libghc-hinotify-prof@0.4-2?arch=arm64", "relation": "produces" }
  ]
}
```

Edge counts: 190 `uses`, 190 `carries_out`, 2 `produces` = **382 total**.
