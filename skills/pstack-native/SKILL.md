---
name: pstack-native
description: Run the six-step native pstack delivery Workflow.
---

# Pstack Native

This directory is one complete Workflow Skill package. The five YAML files beside
this entry define its execution graph; the native pstack Skills, scripts and
resources live in `skills/`.

Select `pstack-native` through the public `fanloop-workflow` entry, initialize with
`fanloop flow init --workflow pstack-native`, and follow the current Status and its
exact required Skill paths. Child Skills remain local to this package.
