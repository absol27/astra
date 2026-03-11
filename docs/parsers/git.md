# Git Parser

## Sample Git Log Input

```text
commit a1b2c3
Author: Alice <alice@example.com>
Date:   Mon Mar 10 10:00:00 2026

    Fix dependency vulnerability

Parsed Output (AStRA Map)
{
  "artifact": "commit:a1b2c3",
  "principal": "alice@example.com",
  "action": "commit",
  "timestamp": "2026-03-10T10:00:00"
}

Notes
Each commit becomes a node in the AStRA artifact graph.
Author is mapped to a principal.

This keeps the **documentation clean and expandable**.

---

## 2️⃣ Alternative: README section (simpler but less scalable)

If the project is small, you could add a section in `README.md`:

```markdown
## Example: Git Log Parsing

Input:

```text
git log --pretty=fuller