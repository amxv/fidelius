#include <gtk/gtk.h>
#include "protocol.h"

typedef struct {
    FILE *out;
    GMainLoop *loop;
    GPtrArray *fields;
    int status;
} Form;

static GtkWidget *label(const char *text) {
    GtkWidget *widget = gtk_label_new(text);
    gtk_label_set_wrap(GTK_LABEL(widget), TRUE);
    gtk_label_set_wrap_mode(GTK_LABEL(widget), PANGO_WRAP_WORD_CHAR);
    gtk_label_set_xalign(GTK_LABEL(widget), 0);
    return widget;
}

static void respond(GtkDialog *dialog, int response, Form *form) {
    if (response == GTK_RESPONSE_ACCEPT) {
        for (guint i = 0; i < form->fields->len; i++) {
            GtkWidget *entry = g_ptr_array_index(form->fields, i);
            if (!gtk_editable_get_text(GTK_EDITABLE(entry))[0]) {
                gtk_widget_add_css_class(entry, "error");
                gtk_widget_grab_focus(entry);
                return;
            }
        }
        form->status = 0;
        for (guint i = 0; i < form->fields->len; i++) {
            const char *value = gtk_editable_get_text(g_ptr_array_index(form->fields, i));
            if (fputs(value, form->out) == EOF || fputc(0, form->out) == EOF)
                form->status = 1;
        }
    }
    g_main_loop_quit(form->loop);
}

static void changed(GtkWidget *entry) { gtk_widget_remove_css_class(entry, "error"); }
static void submit(GtkDialog *dialog) { gtk_dialog_response(dialog, GTK_RESPONSE_ACCEPT); }

int main(void) {
    Form form = {.out = result_stream(), .status = 2};
    char *message = next_string(), *expiry = next_string();
    g_set_prgname("io.github.amxv.Fidelius");
    g_set_application_name("Fidelius");
    if (!form.out || !message || !expiry || !gtk_init_check()) return 1;
    form.loop = g_main_loop_new(NULL, FALSE);
    form.fields = g_ptr_array_new();
    GtkWidget *dialog = gtk_dialog_new_with_buttons("Fidelius", NULL, 0,
        "Cancel", GTK_RESPONSE_CANCEL, "Submit", GTK_RESPONSE_ACCEPT, NULL);
    gtk_window_set_icon_name(GTK_WINDOW(dialog), "io.github.amxv.Fidelius");
    gtk_dialog_set_default_response(GTK_DIALOG(dialog), GTK_RESPONSE_ACCEPT);
    GtkWidget *scroll = gtk_scrolled_window_new();
    gtk_scrolled_window_set_policy(GTK_SCROLLED_WINDOW(scroll), GTK_POLICY_NEVER, GTK_POLICY_AUTOMATIC);
    gtk_widget_set_vexpand(scroll, TRUE);
    gtk_box_append(GTK_BOX(gtk_dialog_get_content_area(GTK_DIALOG(dialog))), scroll);
    GtkWidget *body = gtk_box_new(GTK_ORIENTATION_VERTICAL, 8);
    gtk_widget_set_margin_start(body, 20);
    gtk_widget_set_margin_end(body, 20);
    gtk_widget_set_margin_top(body, 16);
    gtk_widget_set_margin_bottom(body, 16);
    gtk_scrolled_window_set_child(GTK_SCROLLED_WINDOW(scroll), body);
    gtk_box_append(GTK_BOX(body), label(message));
    char *name;
    while ((name = next_string())) {
        GtkWidget *caption = label(name);
        gtk_widget_set_margin_top(caption, 6);
        GtkWidget *entry = gtk_password_entry_new();
        gtk_accessible_update_relation(GTK_ACCESSIBLE(entry), GTK_ACCESSIBLE_RELATION_LABELLED_BY,
                                       caption, NULL, -1);
        gtk_box_append(GTK_BOX(body), caption);
        gtk_box_append(GTK_BOX(body), entry);
        g_ptr_array_add(form.fields, entry);
        g_signal_connect(entry, "changed", G_CALLBACK(changed), NULL);
        g_signal_connect_swapped(entry, "activate", G_CALLBACK(submit), dialog);
        free(name);
    }
    if (!form.fields->len) return 1;
    GtkWidget *hint = label(expiry);
    gtk_widget_add_css_class(hint, "dim-label");
    gtk_box_append(GTK_BOX(body), hint);
    gtk_window_set_default_size(GTK_WINDOW(dialog), 460, 190 + 75 * MIN(form.fields->len, 4));
    g_signal_connect(dialog, "response", G_CALLBACK(respond), &form);
    if (!signal_ready(form.out)) return 1;
    gtk_window_present(GTK_WINDOW(dialog));
    gtk_widget_grab_focus(g_ptr_array_index(form.fields, 0));
    g_main_loop_run(form.loop);
    gtk_window_destroy(GTK_WINDOW(dialog));
    return fclose(form.out) == 0 ? form.status : 1;
}
