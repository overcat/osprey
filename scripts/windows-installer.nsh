; Osprey's own steps in its Windows installer, which mygo.json's
; windows.installerInclude names: the install puts Osprey's folder on the
; user's PATH, as osp install-cli does, so that new terminals run osp and
; osprey, and the uninstall takes it off again. nsExec runs Osprey without
; a console window, and its exit code goes off the stack: a PATH Osprey
; couldn't change doesn't stop the install.

!macro customInstall
  nsExec::Exec '"$INSTDIR\Osprey.exe" install-cli'
  Pop $0
!macroend

!macro customUnInstall
  nsExec::Exec '"$INSTDIR\Osprey.exe" uninstall-cli'
  Pop $0
!macroend
