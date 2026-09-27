Singapore PSI Widget  (SingaporePSI.exe, v1.2.0)
=================================================

A small frameless desktop widget showing a map of Singapore with the latest
24-hour PSI for North/South/East/West/Central, plus 1-hour PM2.5, and a
timeline of the past 48 hours.
Data: data.gov.sg (NEA), refreshed every 15 minutes. Map: OneMap (SLA).

Single file, no installer, no admin rights, no PowerShell, no browser needed.
It uses the Microsoft Edge WebView2 Runtime that is built into Windows 11
(this is separate from the Edge browser).

FIRST RUN
1. Put SingaporePSI.exe somewhere permanent, e.g.
   C:\Users\<you>\Apps\SingaporePSI\SingaporePSI.exe
   (If you move it later, just run it once from the new place.)
2. If you downloaded it: right-click SingaporePSI.exe > Properties >
   tick "Unblock" (bottom of the General tab) > OK.
3. Double-click it. If SmartScreen says "Windows protected your PC", click
   "More info" > "Run anyway" (the exe is not code-signed).
4. The widget appears at the top-right of your main screen.

USING IT
- Drag it anywhere by clicking and dragging on the widget (not on buttons).
  The position is remembered.
- Buttons: A- / A+ = smaller / larger text (90% to 200%, the window grows
  with it and the size is remembered);  refresh = fetch now;  - = minimize;
  X = close.  Keyboard: Ctrl + / Ctrl - / Ctrl 0 (reset), or Ctrl + mouse wheel.
- Timeline (bottom): one bar per hour for the past 48 hours, newest on the
  right; bar colour/number = highest 24-hr PSI of the 5 regions that hour.
  Scroll it with the mouse wheel, by dragging, or with the arrow keys.
  Hover an hour to preview it, click to keep it: the map and header then
  show that hour ("Viewing ... (historical)"). Click "Now" or press Esc to
  return to the latest reading. Auto-refresh keeps your selected hour.
- The widget follows your Windows display scale (Settings > System > Display
  > Scale) automatically, including when dragged to another monitor.
- "Start with Windows" checkbox (bottom right): tick to start the widget
  automatically when you sign in. Untick to stop. (Off by default.)
  This adds/removes a value named "SingaporePSIWidget" under
  HKCU\Software\Microsoft\Windows\CurrentVersion\Run. You can also see or
  disable it in Task Manager > Startup apps.
- Manual alternative instead of the checkbox: press Win+R, type
  shell:startup, press Enter, and put a shortcut to SingaporePSI.exe there.
- Starting it again while it is already running just brings it to the front.
- If offline, it shows the last readings with a "STALE" note.

FILES IT CREATES (per user)
  %LOCALAPPDATA%\PSIWidget\app       - the widget's web page (extracted)
  %LOCALAPPDATA%\PSIWidget\WebView2  - browser cache/storage for the widget
  %APPDATA%\PSIWidget\settings.json  - remembered window position

UNINSTALL
1. Untick "Start with Windows" (or remove the shortcut from shell:startup).
2. Close the widget and delete SingaporePSI.exe.
3. Optionally delete the folders %LOCALAPPDATA%\PSIWidget and %APPDATA%\PSIWidget.

UPDATING FROM AN OLDER VERSION
Close the widget (X), replace SingaporePSI.exe with the new one in the SAME
folder, then start it again. "Start with Windows" keeps working.

DIAGNOSTICS
Run "SingaporePSI.exe --debug" (e.g. from a shortcut or Win+R with the full
path). A yellow line at the bottom shows the detected DPI, DPI awareness,
WebView2 RasterizationScale and ZoomFactor. Right-click / F12 dev tools are
also enabled in this mode.

IF IT DOES NOT START
- "WebView2 Runtime not found" message: install the free Evergreen
  Bootstrapper from https://developer.microsoft.com/microsoft-edge/webview2/
- If your antivirus (e.g. Bitdefender) quarantines it, restore it and add an
  exception for the file. It is an unsigned, freshly built program, which
  is why some antivirus heuristics are cautious.
