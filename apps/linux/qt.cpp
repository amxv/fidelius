#include <QtWidgets>
#include <limits>
#include "protocol.h"

static QLabel *label(const QString &text) {
    auto *caption = new QLabel;
    caption->setTextFormat(Qt::PlainText);
    caption->setWordWrap(true);
    caption->setText(text);
    return caption;
}

int main(int argc, char **argv) {
    FILE *out = result_stream();
    char *message = next_string(), *expiry = next_string();
    if (!out || !message || !expiry) return 1;
    QApplication app(argc, argv);
    app.setDesktopFileName("io.github.amxv.Fidelius");
    app.setApplicationName("Fidelius");
    app.setWindowIcon(QIcon::fromTheme("io.github.amxv.Fidelius"));

    QDialog dialog;
    dialog.setWindowTitle("Fidelius");
    auto *layout = new QVBoxLayout(&dialog);
    auto *scroll = new QScrollArea;
    scroll->setWidgetResizable(true);
    scroll->setFrameShape(QFrame::NoFrame);
    auto *body = new QWidget;
    auto *form = new QVBoxLayout(body);
    form->setAlignment(Qt::AlignTop);
    auto *intro = label(QString::fromUtf8(message));
    intro->setTextInteractionFlags(Qt::TextSelectableByMouse);
    form->addWidget(intro);
    QList<QLineEdit *> fields;
    while (char *raw = next_string()) {
        QString name = QString::fromUtf8(raw);
        free(raw);
        form->addSpacing(6);
        form->addWidget(label(name));
        auto *field = new QLineEdit;
        field->setAccessibleName(name);
        field->setEchoMode(QLineEdit::Password);
        field->setMaxLength(std::numeric_limits<int>::max());
        form->addWidget(field);
        fields.append(field);
    }
    if (fields.isEmpty()) return 1;
    form->addWidget(label(QString::fromUtf8(expiry)));
    scroll->setWidget(body);
    layout->addWidget(scroll);
    auto *buttons = new QDialogButtonBox(QDialogButtonBox::Cancel);
    auto *submit = buttons->addButton("Submit", QDialogButtonBox::AcceptRole);
    buttons->button(QDialogButtonBox::Cancel)->setAutoDefault(false);
    submit->setAutoDefault(true);
    submit->setDefault(true);
    layout->addWidget(buttons);
    QObject::connect(buttons, &QDialogButtonBox::rejected, &dialog, &QDialog::reject);
    QObject::connect(buttons, &QDialogButtonBox::accepted, [&] {
        for (auto *field : fields) {
            if (field->text().isEmpty()) {
                field->setPlaceholderText("Required");
                field->setFocus();
                scroll->ensureWidgetVisible(field);
                return;
            }
        }
        dialog.accept();
    });
    dialog.resize(460, 170 + 70 * qMin<qsizetype>(fields.size(), 4));
    fields.first()->setFocus();
    if (!signal_ready(out)) return 1;
    if (dialog.exec() != QDialog::Accepted) return 2;
    for (auto *field : fields) {
        QByteArray bytes = field->text().toUtf8();
        if (fwrite(bytes.constData(), 1, bytes.size(), out) != size_t(bytes.size()) ||
            fputc(0, out) == EOF) return 1;
    }
    return fclose(out) == 0 ? 0 : 1;
}
