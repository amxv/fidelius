$ErrorActionPreference = 'Stop'
try {
    Add-Type -AssemblyName System.Windows.Forms
    Add-Type -AssemblyName System.Drawing
    [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
    $requestJson = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String([Console]::In.ReadToEnd()))
    $request = ConvertFrom-Json -InputObject $requestJson

    $form = New-Object System.Windows.Forms.Form
    $form.Text = 'Fidelius'
    $form.StartPosition = 'CenterScreen'
    $form.Width = 540
    $form.Height = [Math]::Min(700, 230 + 75 * $request.Names.Count)
    $form.MinimumSize = [System.Drawing.Size]::new(540, 230)
    $form.FormBorderStyle = 'Sizable'
    $form.ShowInTaskbar = $true

    $fields = @{}
    $content = New-Object System.Windows.Forms.FlowLayoutPanel
    $content.Dock = 'Fill'
    $content.AutoScroll = $true
    $content.FlowDirection = 'TopDown'
    $content.WrapContents = $false
    $content.Padding = [System.Windows.Forms.Padding]::new(16)
    $form.Controls.Add($content)

    $message = if ([string]::IsNullOrWhiteSpace($request.Message)) { 'Paste the requested secrets.' } else { $request.Message }
    $description = New-Object System.Windows.Forms.Label
    $description.Text = "$message`r`n`r`nAuto-delete in $($request.AutoDeleteLabel)."
    $description.AutoSize = $true
    $description.MaximumSize = [System.Drawing.Size]::new(465, 0)
    $description.Margin = [System.Windows.Forms.Padding]::new(0, 0, 0, 15)
    $content.Controls.Add($description)

    foreach ($name in $request.Names) {
        $label = New-Object System.Windows.Forms.Label
        $label.Text = $name
        $label.Width = 465
        $label.Height = 20
        $label.Margin = [System.Windows.Forms.Padding]::new(0, 4, 0, 2)
        $content.Controls.Add($label)

        $input = New-Object System.Windows.Forms.TextBox
        $input.Width = 465
        $input.UseSystemPasswordChar = $true
        $input.Margin = [System.Windows.Forms.Padding]::new(0, 0, 0, 12)
        $content.Controls.Add($input)
        $fields[$name] = $input
    }

    $buttons = New-Object System.Windows.Forms.FlowLayoutPanel
    $buttons.Dock = 'Bottom'
    $buttons.FlowDirection = 'RightToLeft'
    $buttons.Height = 54
    $buttons.Padding = [System.Windows.Forms.Padding]::new(8)
    $form.Controls.Add($buttons)
    $buttons.BringToFront()

    $save = New-Object System.Windows.Forms.Button
    $save.Text = 'Save'
    $save.Width = 90
    $save.Add_Click({
        foreach ($name in $request.Names) {
            if ($fields[$name].Text.Length -eq 0) {
                [System.Windows.Forms.MessageBox]::Show($form, 'Every secret is required.', 'Fidelius') | Out-Null
                $fields[$name].Focus() | Out-Null
                return
            }
        }
        $form.DialogResult = [System.Windows.Forms.DialogResult]::OK
        $form.Close()
    })
    $buttons.Controls.Add($save)
    $form.AcceptButton = $save

    $cancel = New-Object System.Windows.Forms.Button
    $cancel.Text = 'Cancel'
    $cancel.Width = 90
    $cancel.DialogResult = [System.Windows.Forms.DialogResult]::Cancel
    $buttons.Controls.Add($cancel)
    $form.CancelButton = $cancel

    $result = $form.ShowDialog()
    if ($result -eq [System.Windows.Forms.DialogResult]::OK) {
        $values = @{}
        foreach ($name in $request.Names) { $values[$name] = $fields[$name].Text }
        [Console]::Out.Write((ConvertTo-Json -InputObject @{ cancelled = $false; values = $values } -Compress -Depth 4))
    } else {
        [Console]::Out.Write('{"cancelled":true}')
    }
    $form.Dispose()
} catch {
    exit 1
}
