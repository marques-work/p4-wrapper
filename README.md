# p4-wrapper

This is a configurable wrapper for the `p4` executable that offers the following capabilities:

  - Works on Windows, Linux, and MacOS X
  - Adds (very) verbose output to each `p4` command (internally prepends `-v 4` to arguments)
  - Logs the environment, time, duration, and exit status
  - Optional output truncation to STDOUT (the log file retains the full content)
  - Is configurable via a `p4-wrapper.json` file in the CWD

## Installation

1. Put this executable in a directory on your `PATH` that precedes the realy `p4` executable.
2. Make sure the name of this executable is also `p4` (`p4.exe` on Windows, naturally)
3. The path to the real `p4` defaults to `C:\Program Files\Perforce\p4.exe` in Windows and `/usr/local/bin/p4` everywhere else. If you need to change this, see below.

## Configuration

Configuration (optional if defaults are ok) is read from a `p4-wrapper.json` file sitting in the **CWD**. The format (and configurable options) are:

```javascript
{
  "p4Path": "/usr/local/bin/p4", // "C:\Program Files\Perforce\p4.exe" on Windows
  "logDir": "/tmp", // "C:\tmp" on Windows
  "verbose": false, // setting true adds `-v 4` to each command
  "maxLines": -1 // Any value > 0 will cause STDOUT to be truncated to the specified `maxLines`; a value of -1 yields the full output
}
```
