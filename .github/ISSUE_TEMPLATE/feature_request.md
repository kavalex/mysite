name: Feature request
description: Suggest an enhancement or a new recipe.
labels: ["enhancement"]
body:
  - type: markdown
    attributes:
      content: |
        Describe the feature, the problem it solves, and the proposed shape.
        See [docs/architecture.md](../../docs/architecture.md) for the locked-in
        decisions and [docs/recipes/](../../docs/recipes/) for existing recipes.

  - type: textarea
    id: problem
    attributes:
      label: Problem
      description: "What goes wrong today? What is the workflow missing?"
    validations:
      required: true

  - type: textarea
    id: proposal
    attributes:
      label: Proposed shape
      description: "High-level design. Code snippets are welcome but keep them illustrative."
    validations:
      required: true

  - type: textarea
    id: alternatives
    attributes:
      label: Alternatives considered
      description: What else did you look at? Why is this better?

  - type: textarea
    id: dependencies
    attributes:
      label: Dependencies
      description: "Does this require a 4th dependency? Why is stdlib not enough? See [ADR-0001](../../docs/adr/0001-stdlib-first.md)."

  - type: checkboxes
    id: checklist
    attributes:
      label: Pre-flight
      options:
        - label: I searched existing issues
          required: true
        - label: This is consistent with [architecture.md](../../docs/architecture.md)
          required: true
