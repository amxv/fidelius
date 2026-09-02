import Foundation

enum FideliusKeychainError: LocalizedError {
    case launch(String)
    case securityTool(account: String, status: Int32, detail: String)

    var errorDescription: String? {
        switch self {
        case let .launch(message):
            return "Could not open macOS Keychain: \(message)"
        case let .securityTool(account, status, detail):
            if detail.isEmpty {
                return "Could not save \(account) to macOS Keychain (security exited \(status))."
            }
            return "Could not save \(account) to macOS Keychain: \(detail)"
        }
    }
}

func saveAPIKey(service: String, account: String, value: String) throws {
    let process = Process()
    process.executableURL = URL(fileURLWithPath: "/usr/bin/security")
    process.arguments = [
        "add-generic-password",
        "-U",
        "-s", service,
        "-a", account,
        "-w",
    ]

    let input = Pipe()
    let output = Pipe()
    let errors = Pipe()
    process.standardInput = input
    process.standardOutput = output
    process.standardError = errors

    do {
        try process.run()
    } catch {
        throw FideliusKeychainError.launch(error.localizedDescription)
    }

    // `security ... -w` reads the password from stdin when -w is the final
    // option. New items ask twice for confirmation; updates consume only what
    // they need. The value never appears in argv, stdout, or stderr.
    let payload = Data("\(value)\n\(value)\n".utf8)
    input.fileHandleForWriting.write(payload)
    input.fileHandleForWriting.closeFile()
    process.waitUntilExit()

    guard process.terminationStatus == 0 else {
        let data = errors.fileHandleForReading.readDataToEndOfFile()
        let detail = String(data: data, encoding: .utf8)?
            .split(separator: "\n")
            .last
            .map(String.init)
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) } ?? ""
        throw FideliusKeychainError.securityTool(
            account: account,
            status: process.terminationStatus,
            detail: detail
        )
    }
}
