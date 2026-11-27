# Accolade 0.14.1

0.14.1 is 0.14.0 with a fix to how the release is built. **v0.14.0 has no downloads:** its macOS Intel build failed, so no binaries were published for it. Everything 0.14.0 brought is here; see [the 0.14.0 notes](RELEASE-0.14.0.md):

- **Scripts from other apps:** open Fade In (`.fadein`, `.osf`), Trelby (`.trelby`) and KIT Scenarist (`.kitsp`) scripts, and save a copy back in their format.
- **Trelby keeps your emphasis and dual dialogue**, written as Fountain's own markers.
- **Adapting across languages** is now measured, and an intermittent hang in the window tests is gone.

## The release fix

The release builds install Fyne's packaging tool fresh each time, and always took its newest version. When a new one came out, the macOS Intel build compiled the tool with the Intel-only compiler flags meant for Accolade itself, and the compiler stopped at two conflicting architectures. The tool now builds without those flags, and every platform uses the same fixed version of it (fyne.io/tools v1.7.4, the one 0.13.0 was built with), so a new version can no longer break a release by surprise.
