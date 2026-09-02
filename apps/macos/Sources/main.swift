import AppKit
import Foundation

func parseRequest() throws -> FideliusRequest {
    let args = Array(CommandLine.arguments.dropFirst())
    var message: String?
    var autoDeleteLabel = "5 minutes"
    var secretFD: Int32?
    var accounts: [String] = []
    var index = 0

    while index < args.count {
        let arg = args[index]
        switch arg {
        case "--message":
            guard index + 1 < args.count else {
                throw NSError(domain: "Fidelius", code: 64, userInfo: [NSLocalizedDescriptionKey: "--message requires a value"])
            }
            index += 1
            message = args[index]
        case "--auto-delete":
            guard index + 1 < args.count else {
                throw NSError(domain: "Fidelius", code: 64, userInfo: [NSLocalizedDescriptionKey: "--auto-delete requires a value"])
            }
            index += 1
            autoDeleteLabel = args[index]
        case "--secret-fd":
            guard index + 1 < args.count, let parsed = Int32(args[index + 1]) else {
                throw NSError(domain: "Fidelius", code: 64, userInfo: [NSLocalizedDescriptionKey: "--secret-fd requires a file descriptor"])
            }
            index += 1
            secretFD = parsed
        default:
            accounts.append(arg)
        }
        index += 1
    }

    guard let secretFD, !accounts.isEmpty else {
        throw NSError(domain: "Fidelius", code: 64, userInfo: [NSLocalizedDescriptionKey: "missing secret pipe or secret names"])
    }
    return FideliusRequest(
        message: message,
        accounts: accounts,
        autoDeleteLabel: autoDeleteLabel,
        secretFD: secretFD
    )
}

do {
    let request = try parseRequest()
    let app = NSApplication.shared
    let delegate = AppDelegate(request: request)
    app.setActivationPolicy(.regular)
    app.delegate = delegate
    app.run()
} catch {
    fputs("\(error.localizedDescription)\n", stderr)
    exit(64)
}
