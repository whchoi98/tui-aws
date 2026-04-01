# Go Conventions

- Use `go vet ./...` before committing
- Error handling: always check and return errors, never ignore with `_`
- Pointer receivers on stateful structs (Favorites, History, EC2Model)
- Value receivers on RootModel (Bubble Tea requirement)
- Naming: CamelCase for exported, camelCase for unexported
- Package names: lowercase, single word when possible
- Imports: stdlib first, then external, then internal (goimports handles this)
