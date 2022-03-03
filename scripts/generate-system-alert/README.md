# Update alert rules in installer.

If you update `system-alert.yaml` or related functions in code, you should run:

```
go run scripts/generate-system-alert/main.go
```