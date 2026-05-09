# Bash Tools

```yaml
tools:
  - type: bash
    name: run_bash
    description: Executa comandos bash permitidos para diagnóstico.
    allowed_commands:
      - echo
      - pwd
      - ls
    timeout_seconds: 20
    max_output_bytes: 8192
```
