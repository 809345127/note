# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a VS Code extension called "Go Implementation to Interface" that helps Go developers navigate from method implementations to their interface definitions. The extension is written in TypeScript with a Go tool for code analysis.

## Commands

### Build and Development
```bash
# Install dependencies
npm install

# Compile TypeScript to JavaScript
npm run compile

# Watch mode for development
npm run watch

# Run tests
npm run test
```

### Packaging
```bash
# Install vsce globally first
npm install -g vsce

# Create VSIX package
vsce package
```

### Debugging
1. Open the project in VS Code
2. Press `F5` to launch the Extension Development Host
3. Test the extension in the new VS Code window
4. Use VS Code's Debug Console for output and errors

## Architecture

### Core Components

1. **TypeScript Extension (`src/extension.ts`)**
   - Main entry point for the VS Code extension
   - Registers command `go-impl-to-intf.findInterface`
   - Provides CodeLens for Go method implementations
   - Handles editor context and user interactions
   - Executes the Go tool as a child process

2. **Go Analysis Tool (`src/impl_to_intf.go`)**
   - Standalone Go program for code analysis
   - Parses Go files using `go/ast` and `go/parser`
   - Finds method signatures at cursor positions
   - Scans workspace for matching interface definitions
   - Outputs location information for VS Code navigation

### Key Integration Points

- **Command Registration**: Defined in `package.json` with activation events for Go language
- **Context Menus**: Right-click menu for Go editors (editor/context)
- **CodeLens**: Inline links above method implementations
- **Go Tool Integration**: Temporary file handling for `go run` execution

### Data Flow
1. User clicks CodeLens or uses context menu
2. Extension captures current file position
3. Go tool is executed with position parameters
4. Go tool analyzes method signature and searches interfaces
5. Extension opens matching interface definition

## Development Notes

### Extension Context
- Activation events: `onLanguage:go` and `onCommand:go-impl-to-intf.findInterface`
- Extension path management for locating Go tool file
- Workspace root detection for Go module context

### Go Tool Execution
- Temporary file creation to avoid `go run` path restrictions
- Cleanup of temporary files after execution
- Error handling and user feedback for Go tool failures

### Performance Considerations
- CodeLens resolution is deferred until visible
- Go tool runs on demand, not cached
- Workspace scanning for interfaces occurs per request

## Testing
- Test runner configured with `vscode-test`
- Tests should verify command execution and Go tool integration
- Mock Go tool responses for unit tests

## Dependencies
- `@types/vscode`: VS Code API type definitions
- `typescript`: TypeScript compiler
- `vscode-test`: Extension testing utilities
- Go language runtime (external dependency for users)