import * as vscode from 'vscode';
import * as child_process from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

// CodeLens provider for Go method implementations
class GoImplToIntfCodeLensProvider implements vscode.CodeLensProvider {
    private _onDidChangeCodeLenses: vscode.EventEmitter<void> = new vscode.EventEmitter<void>();
    public readonly onDidChangeCodeLenses: vscode.Event<void> = this._onDidChangeCodeLenses.event;
    private context: vscode.ExtensionContext;

    constructor(context: vscode.ExtensionContext) {
        this.context = context;
        // Refresh CodeLenses when document changes or saved
        vscode.workspace.onDidChangeTextDocument((e) => {
            // Only refresh if the changed document is a Go file to improve performance
            if (e.document.languageId === 'go') {
                this._onDidChangeCodeLenses.fire();
            }
        });
        
        // Also refresh when document is saved
        vscode.workspace.onDidSaveTextDocument((e) => {
            if (e.languageId === 'go') {
                this._onDidChangeCodeLenses.fire();
            }
        });
    }

    provideCodeLenses(document: vscode.TextDocument, _token: vscode.CancellationToken): vscode.ProviderResult<vscode.CodeLens[]> {
        // Only provide CodeLenses for Go files
        if (document.languageId !== 'go') {
            return [];
        }

        const codeLenses: vscode.CodeLens[] = [];
        
        // Only add CodeLens to methods (functions with receivers)
        // This excludes regular functions without receivers
        const text = document.getText();
        const funcRegex = /func\s+(\([^)]+\)\s+)[\w]+\s*\([^)]*\)/g;
        let match;
        
        while ((match = funcRegex.exec(text)) !== null) {
            const startPos = document.positionAt(match.index);
            const range = new vscode.Range(startPos, startPos);
            
            // Create CodeLens for this function
            const codeLens = new vscode.CodeLens(range, {
                command: 'go-impl-to-intf.findInterface',
                title: 'Find Interface',
                arguments: [startPos.line + 1, startPos.character + 1] // Pass line and column
            });
            
            codeLenses.push(codeLens);
        }
        
        return codeLenses;
    }

    resolveCodeLens(codeLens: vscode.CodeLens, _token: vscode.CancellationToken): vscode.ProviderResult<vscode.CodeLens> {
        // 只在方法可见时才检查是否有对应的接口
        // 这样可以提高性能，避免在文档加载时检查所有方法
        
        // 获取当前活动编辑器
        const editor = vscode.window.activeTextEditor;
        if (!editor) {
            return null;
        }

        const document = editor.document;
        const args = codeLens.command?.arguments;
        if (!args || args.length < 2) {
            return null;
        }

        const lineNumber = args[0] as number;
        const columnNumber = args[1] as number;
        
        // 检查是否有对应的接口 - 总是使用本地工具检查，避免gopls缓存问题
        return this.checkInterfaceWithTool(document, lineNumber, columnNumber).then((hasInterface) => {
            return hasInterface ? codeLens : null;
        });
    }
    
    // 执行impl_to_intf工具并获取结果
    public executeImplToIntfTool(filePath: string, lineNumber: number, columnNumber: number, showError: boolean = false): Promise<string | null> {
        return new Promise<string | null>((resolve) => {
            // 获取扩展根目录
            const extensionRoot = this.context.extensionPath;
            if (!extensionRoot) {
                resolve(null);
                return;
            }

            // 找到项目根目录
            const workspaceFolders = vscode.workspace.workspaceFolders;
            if (!workspaceFolders) {
                resolve(null);
                return;
            }

            const workspaceRoot = workspaceFolders[0].uri.fsPath;
            
            // 检查impl_to_intf.go文件是否存在
            const toolPath = path.join(extensionRoot, 'src', 'impl_to_intf.go');
            if (!fs.existsSync(toolPath)) {
                console.error('impl_to_intf.go file not found at:', toolPath);
                if (showError) {
                    vscode.window.showErrorMessage('impl_to_intf.go file not found. Please reinstall the extension.');
                }
                resolve(null);
                return;
            }
            
            // 执行go run命令，需要先复制工具文件到项目根目录，因为go run要求文件在同一目录
            // 使用唯一文件名避免竞态条件
            const tempToolPath = path.join(workspaceRoot, `impl_to_intf_temp_${Date.now()}_${Math.random().toString(36).substring(2, 9)}.go`);
            const tempFileName = path.basename(tempToolPath);
            
            try {
                fs.copyFileSync(toolPath, tempToolPath);
            } catch (err) {
                console.error('Error copying temporary file:', err);
                if (showError) {
                    vscode.window.showErrorMessage(`Error copying tool file: ${err instanceof Error ? err.message : String(err)}`);
                }
                resolve(null);
                return;
            }
            
            // 合并为一个格式为 <file>:<line>:<col> 的参数，这是程序期望的格式
            const positionParam = `${filePath}:${lineNumber}:${columnNumber}`;
            const cmd = `go run "${tempFileName}" "${positionParam}"`;
            
            child_process.exec(cmd, { cwd: workspaceRoot }, (error, stdout, stderr) => {
                // 无论执行结果如何，都要清理临时文件
                try {
                    if (fs.existsSync(tempToolPath)) {
                        fs.unlinkSync(tempToolPath);
                    }
                } catch (err) {
                    console.error('Error deleting temporary file:', err);
                }

                if (error) {
                    console.error('Error executing impl_to_intf tool:', stderr);
                    if (showError) {
                        vscode.window.showErrorMessage(`Error: ${error.message}`);
                    }
                    resolve(null);
                    return;
                }

                resolve(stdout);
            });
        });
    }

    // 检查方法是否实现了接口
    private checkInterfaceWithTool(document: vscode.TextDocument, lineNumber: number, columnNumber: number): Promise<boolean> {
        return new Promise<boolean>((resolve) => {
            const filePath = document.uri.fsPath;
            
            this.executeImplToIntfTool(filePath, lineNumber, columnNumber, false).then((stdout) => {
                if (!stdout) {
                    resolve(false);
                    return;
                }

                // 检查输出中是否包含JSON_START
                if (stdout.indexOf('JSON_START') !== -1) {
                    resolve(true); // 找到接口
                } else {
                    resolve(false); // 没有找到接口
                }
            });
        });
    }
}

export function activate(context: vscode.ExtensionContext) {
    // 注册命令
    let disposable = vscode.commands.registerCommand('go-impl-to-intf.findInterface', (lineNumber?: number, columnNumber?: number) => {
        // 获取当前活动编辑器
        const editor = vscode.window.activeTextEditor;
        if (!editor) {
            vscode.window.showErrorMessage('No active editor found');
            return;
        }

        // 获取当前文件路径
        const filePath = editor.document.uri.fsPath;
        
        // 如果没有提供行号和列号，则使用当前光标位置
        if (!lineNumber || !columnNumber) {
            const position = editor.selection.active;
            lineNumber = position.line + 1; // VS Code行号从0开始，我们的工具从1开始
            columnNumber = position.character + 1;
        }

        // 找到项目根目录
        const workspaceFolders = vscode.workspace.workspaceFolders;
        if (!workspaceFolders) {
            vscode.window.showErrorMessage('No workspace folder found');
            return;
        }

        const workspaceRoot = workspaceFolders[0].uri.fsPath;
        
        // 创建临时的CodeLensProvider实例来使用其工具执行方法
        const codeLensProvider = new GoImplToIntfCodeLensProvider(context);
        
        // 执行impl_to_intf工具获取接口信息
        codeLensProvider.executeImplToIntfTool(filePath, lineNumber, columnNumber, true).then((stdout) => {
            if (stdout) {
                // 解析Go工具输出的JSON信息
                const jsonStart = stdout.indexOf('JSON_START');
                const jsonEnd = stdout.indexOf('JSON_END');
                
                if (jsonStart !== -1 && jsonEnd !== -1) {
                    const jsonStr = stdout.substring(jsonStart + 10, jsonEnd);
                    try {
                        const result = JSON.parse(jsonStr);
                        const { file, line, interface: interfaceName } = result;
                        
                        // 使用VSCode API打开文件并定位到指定方法行
                        const fileUri = vscode.Uri.file(file);
                        // 优先使用方法行号，如果没有则使用接口行号
                        const targetLine = result.methodLine || line;
                        const options: vscode.TextDocumentShowOptions = {
                            selection: new vscode.Range(
                                new vscode.Position(targetLine - 1, 0),
                                new vscode.Position(targetLine - 1, 0)
                            ),
                            viewColumn: vscode.ViewColumn.Beside
                        };
                        
                        vscode.window.showTextDocument(fileUri, options).then(() => {
                            // 如果找到了具体方法，显示更详细的信息
                            if (result.methodLine && result.method) {
                                vscode.window.showInformationMessage(
                                    `Found interface "${interfaceName}" with method "${result.method}" at line ${result.methodLine}`
                                );
                            } else {
                                vscode.window.showInformationMessage(
                                    `Found interface "${interfaceName}" at line ${line}`
                                );
                            }
                        }, err => {
                            vscode.window.showErrorMessage(
                                `Failed to open file: ${err.message}`
                            );
                        });
                    } catch (parseError) {
                        console.error('Error parsing JSON:', parseError);
                        vscode.window.showErrorMessage('Failed to parse interface information');
                    }
                } else {
                    // 如果没有找到JSON信息，显示普通输出
                    vscode.window.showInformationMessage(stdout.trim());
                }
            }
        });
    });

    context.subscriptions.push(disposable);

    // Register CodeLens provider
    const codeLensProvider = new GoImplToIntfCodeLensProvider(context);
    const codeLensDisposable = vscode.languages.registerCodeLensProvider('go', codeLensProvider);
    context.subscriptions.push(codeLensDisposable);
}

export function deactivate() {}
