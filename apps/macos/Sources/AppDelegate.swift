import AppKit
import Foundation

struct FideliusRequest {
    let service: String
    let accounts: [String]
}

private struct SavedKey: Codable {
    let account: String
    let length: Int
}

private struct PromptResponse: Codable {
    let cancelled: Bool
    let saved: [SavedKey]
}

final class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate, NSTextFieldDelegate {
    private let request: FideliusRequest
    private var window: NSWindow?
    private var fields: [String: NSSecureTextField] = [:]
    private var saveButton: NSButton?
    private var didFinish = false

    init(request: FideliusRequest) {
        self.request = request
        super.init()
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        buildMenus()
        showWindow()
    }

    func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply {
        if !didFinish {
            emit(PromptResponse(cancelled: true, saved: []))
            didFinish = true
        }
        return .terminateNow
    }

    func windowWillClose(_ notification: Notification) {
        cancel(nil)
    }

    func controlTextDidChange(_ obj: Notification) {
        updateSaveButton()
    }

    @objc private func save(_ sender: Any?) {
        let missing = request.accounts.first { account in
            guard let field = fields[account] else { return true }
            return field.stringValue.isEmpty
        }
        if let missing, let field = fields[missing] {
            NSSound.beep()
            window?.makeFirstResponder(field)
            return
        }

        do {
            var saved: [SavedKey] = []
            for account in request.accounts {
                guard let value = fields[account]?.stringValue else { continue }
                try saveAPIKey(service: request.service, account: account, value: value)
                saved.append(SavedKey(account: account, length: value.count))
            }
            finish(PromptResponse(cancelled: false, saved: saved))
        } catch {
            showError(error.localizedDescription)
        }
    }

    @objc private func cancel(_ sender: Any?) {
        guard !didFinish else { return }
        finish(PromptResponse(cancelled: true, saved: []))
    }

    private func finish(_ response: PromptResponse) {
        guard !didFinish else { return }
        didFinish = true
        emit(response)
        NSApp.terminate(nil)
    }

    private func emit(_ response: PromptResponse) {
        guard let data = try? JSONEncoder().encode(response) else { return }
        FileHandle.standardOutput.write(data)
        FileHandle.standardOutput.write(Data("\n".utf8))
    }

    private func updateSaveButton() {
        saveButton?.isEnabled = request.accounts.allSatisfy { account in
            !(fields[account]?.stringValue.isEmpty ?? true)
        }
    }

    private func showError(_ message: String) {
        let alert = NSAlert()
        alert.alertStyle = .warning
        alert.messageText = "Fidelius couldn’t save the API keys"
        alert.informativeText = message
        if let window {
            alert.beginSheetModal(for: window)
        } else {
            alert.runModal()
        }
    }

    private func showWindow() {
        let count = request.accounts.count
        let height = min(620, max(330, 240 + count * 72))
        let frame = NSRect(x: 0, y: 0, width: 520, height: height)
        let window = NSWindow(
            contentRect: frame,
            styleMask: [.titled, .closable, .miniaturizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Fidelius"
        window.titlebarAppearsTransparent = true
        window.isMovableByWindowBackground = true
        window.level = .normal
        window.delegate = self
        window.isReleasedWhenClosed = false
        window.center()
        self.window = window

        let root = NSStackView()
        root.orientation = .vertical
        root.alignment = .leading
        root.spacing = 14
        root.translatesAutoresizingMaskIntoConstraints = false

        let symbol = NSImageView()
        symbol.image = NSImage(systemSymbolName: "key.fill", accessibilityDescription: "Key")
        symbol.symbolConfiguration = NSImage.SymbolConfiguration(pointSize: 24, weight: .medium)
        symbol.contentTintColor = .secondaryLabelColor
        root.addArrangedSubview(symbol)

        let title = NSTextField(labelWithString: count == 1 ? "An agent needs an API key." : "An agent needs \(count) API keys.")
        title.font = .systemFont(ofSize: 22, weight: .semibold)
        root.addArrangedSubview(title)

        let detail = wrappingLabel(
            "Paste the requested values below. Fidelius will save them to macOS Keychain under service “\(request.service)” and then close."
        )
        detail.textColor = .secondaryLabelColor
        root.addArrangedSubview(detail)

        let scrollView = NSScrollView()
        scrollView.drawsBackground = false
        scrollView.hasVerticalScroller = count > 5
        scrollView.borderType = .noBorder
        scrollView.translatesAutoresizingMaskIntoConstraints = false

        let fieldStack = NSStackView()
        fieldStack.orientation = .vertical
        fieldStack.alignment = .leading
        fieldStack.spacing = 13
        fieldStack.translatesAutoresizingMaskIntoConstraints = false

        for account in request.accounts {
            let group = NSStackView()
            group.orientation = .vertical
            group.alignment = .leading
            group.spacing = 6
            group.translatesAutoresizingMaskIntoConstraints = false

            let label = NSTextField(labelWithString: account)
            label.font = .monospacedSystemFont(ofSize: 12, weight: .medium)
            label.textColor = .secondaryLabelColor

            let field = NSSecureTextField()
            field.placeholderString = "Paste API key"
            field.font = .systemFont(ofSize: 14)
            field.delegate = self
            field.translatesAutoresizingMaskIntoConstraints = false
            field.heightAnchor.constraint(equalToConstant: 34).isActive = true
            field.widthAnchor.constraint(equalToConstant: 472).isActive = true
            fields[account] = field

            group.addArrangedSubview(label)
            group.addArrangedSubview(field)
            fieldStack.addArrangedSubview(group)
        }

        let documentView = NSView()
        documentView.translatesAutoresizingMaskIntoConstraints = false
        documentView.addSubview(fieldStack)
        NSLayoutConstraint.activate([
            fieldStack.leadingAnchor.constraint(equalTo: documentView.leadingAnchor),
            fieldStack.trailingAnchor.constraint(equalTo: documentView.trailingAnchor),
            fieldStack.topAnchor.constraint(equalTo: documentView.topAnchor),
            fieldStack.bottomAnchor.constraint(equalTo: documentView.bottomAnchor),
            documentView.widthAnchor.constraint(equalToConstant: 472),
        ])
        scrollView.documentView = documentView
        root.addArrangedSubview(scrollView)
        scrollView.widthAnchor.constraint(equalToConstant: 472).isActive = true
        scrollView.heightAnchor.constraint(equalToConstant: min(CGFloat(count * 66), 300)).isActive = true

        let buttonRow = NSStackView()
        buttonRow.orientation = .horizontal
        buttonRow.alignment = .centerY
        buttonRow.spacing = 8

        let spacer = NSView()
        spacer.translatesAutoresizingMaskIntoConstraints = false
        buttonRow.addArrangedSubview(spacer)

        let cancelButton = NSButton(title: "Cancel", target: self, action: #selector(cancel(_:)))
        cancelButton.keyEquivalent = "\u{1b}"
        buttonRow.addArrangedSubview(cancelButton)

        let saveTitle = count == 1 ? "Save Key" : "Save Keys"
        let saveButton = NSButton(title: saveTitle, target: self, action: #selector(save(_:)))
        saveButton.keyEquivalent = "\r"
        saveButton.bezelStyle = .rounded
        saveButton.isEnabled = false
        self.saveButton = saveButton
        buttonRow.addArrangedSubview(saveButton)

        root.addArrangedSubview(buttonRow)
        buttonRow.widthAnchor.constraint(equalToConstant: 472).isActive = true

        let content = NSView()
        content.addSubview(root)
        NSLayoutConstraint.activate([
            root.leadingAnchor.constraint(equalTo: content.leadingAnchor, constant: 24),
            root.trailingAnchor.constraint(equalTo: content.trailingAnchor, constant: -24),
            root.topAnchor.constraint(equalTo: content.topAnchor, constant: 24),
            root.bottomAnchor.constraint(lessThanOrEqualTo: content.bottomAnchor, constant: -20),
        ])
        window.contentView = content

        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        if let first = request.accounts.first, let field = fields[first] {
            window.makeFirstResponder(field)
        }
    }

    private func wrappingLabel(_ text: String) -> NSTextField {
        let label = NSTextField(wrappingLabelWithString: text)
        label.font = .systemFont(ofSize: 13)
        label.maximumNumberOfLines = 3
        label.preferredMaxLayoutWidth = 472
        return label
    }

    private func buildMenus() {
        let mainMenu = NSMenu()

        let appItem = NSMenuItem()
        mainMenu.addItem(appItem)
        let appMenu = NSMenu()
        appMenu.addItem(withTitle: "Quit Fidelius", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        appItem.submenu = appMenu

        let editItem = NSMenuItem()
        mainMenu.addItem(editItem)
        let editMenu = NSMenu(title: "Edit")
        editMenu.addItem(withTitle: "Undo", action: Selector(("undo:")), keyEquivalent: "z")
        editMenu.addItem(NSMenuItem.separator())
        editMenu.addItem(withTitle: "Cut", action: #selector(NSText.cut(_:)), keyEquivalent: "x")
        editMenu.addItem(withTitle: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c")
        editMenu.addItem(withTitle: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v")
        editMenu.addItem(withTitle: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a")
        editItem.submenu = editMenu

        NSApp.mainMenu = mainMenu
    }
}
