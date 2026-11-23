# Drive only the windows owned by the synthetic test process.
$ErrorActionPreference = 'Stop'
Add-Type @'
using System;
using System.Text;
using System.Runtime.InteropServices;
public static class DialogSmoke {
    public delegate bool Visitor(IntPtr hwnd, IntPtr data);
    [DllImport("user32.dll")] public static extern bool EnumWindows(Visitor visit, IntPtr data);
    [DllImport("user32.dll")] public static extern bool EnumChildWindows(IntPtr parent, Visitor visit, IntPtr data);
    [DllImport("user32.dll")] public static extern uint GetWindowThreadProcessId(IntPtr hwnd, out int pid);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr hwnd, StringBuilder text, int size);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetClassName(IntPtr hwnd, StringBuilder text, int size);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern IntPtr SendMessage(IntPtr hwnd, uint msg, IntPtr wParam, string value);
    [DllImport("user32.dll")] public static extern bool PostMessage(IntPtr hwnd, uint msg, IntPtr wParam, IntPtr lParam);
    public static bool Submit(int pid) {
        IntPtr dialog=IntPtr.Zero, field=IntPtr.Zero, submit=IntPtr.Zero;
        var text = new StringBuilder(256);
        EnumWindows((h, unused) => {
            int owner;
            GetWindowThreadProcessId(h, out owner);
            if (owner == pid && GetWindowText(h,text,256)>0 && text.ToString()=="Fidelius") dialog=h;
            return true;
        }, IntPtr.Zero);
        if(dialog==IntPtr.Zero) return false;
        EnumChildWindows(dialog, (h, unused) => {
            GetClassName(h,text,256);
            if(text.ToString().Equals("Edit", StringComparison.OrdinalIgnoreCase)) field=h;
            if(GetWindowText(h,text,256)>0 && text.ToString()=="Submit") submit=h;
            return true;
        }, IntPtr.Zero);
        if(field==IntPtr.Zero || submit==IntPtr.Zero) return false;
        SendMessage(field,0x000c,IntPtr.Zero,"dummy-smoke-value");
        return PostMessage(submit,0x00f5,IntPtr.Zero,IntPtr.Zero);
    }
}
'@
$info = [Diagnostics.ProcessStartInfo]::new((Resolve-Path dist/fidelius-windows-amd64.exe).Path, 'ask TOKEN')
$info.UseShellExecute = $false
$info.RedirectStandardOutput = $true
$info.RedirectStandardError = $true
$process = [Diagnostics.Process]::Start($info)
try {
    $deadline = [DateTime]::UtcNow.AddSeconds(20)
    while (!$process.HasExited -and [DateTime]::UtcNow -lt $deadline) {
        if ([DialogSmoke]::Submit($process.Id)) { break }
        Start-Sleep -Milliseconds 100
    }
    if (!$process.WaitForExit(5000)) { throw 'The Win32 dialog did not submit' }
    $stderr = $process.StandardError.ReadToEnd()
    if ($process.ExitCode -ne 0) { throw "Win32 prompt failed (exit $($process.ExitCode))" }
    $path = $process.StandardOutput.ReadToEnd().TrimEnd()
    if ($stderr.Contains('dummy-smoke-value') -or $path.Contains('dummy-smoke-value')) { throw 'Value in CLI output' }
    if ([IO.File]::ReadAllText((Join-Path $path TOKEN)) -cne 'dummy-smoke-value') { throw 'Wrong session contents' }
} finally {
    if (!$process.HasExited) { $process.Kill() }
    $process.Dispose()
}
