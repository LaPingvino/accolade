{
  description = "Accolade - A distraction-free Fountain editor";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        
        # Fyne dependencies
        fyneLibs = with pkgs; [
          xorg.libX11
          xorg.libXcursor
          xorg.libXrandr
          xorg.libXinerama
          xorg.libXi
          xorg.libXext
          xorg.libXfixes
          mesa
          libGL
          alsa-lib
          pkg-config
        ];
        
        # Build tools
        buildInputs = with pkgs; [
          go
          gcc
          pkg-config
        ] ++ fyneLibs;
        
        # Runtime dependencies
        nativeBuildInputs = with pkgs; [
          pkg-config
        ];
        
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = buildInputs;
          nativeBuildInputs = nativeBuildInputs;
          
          shellHook = ''
            export CGO_ENABLED=1
            export PKG_CONFIG_PATH="${pkgs.lib.makeSearchPath "lib/pkgconfig" fyneLibs}"
            export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath fyneLibs}:$LD_LIBRARY_PATH"
            
            echo "Accolade development environment activated"
            echo "Go version: $(go version)"
            echo "Fyne development libraries are available"
            echo ""
            echo "To build: go build -v ."
            echo "To run: ./accolade"
            echo "To package: fyne package -os linux"
          '';
        };
        
        packages.default = pkgs.buildGoModule {
          pname = "accolade";
          version = "0.1.0";
          src = ./.;
          
          vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
          
          buildInputs = fyneLibs;
          nativeBuildInputs = nativeBuildInputs;
          
          CGO_ENABLED = 1;
          
          preBuild = ''
            export PKG_CONFIG_PATH="${pkgs.lib.makeSearchPath "lib/pkgconfig" fyneLibs}"
          '';
          
          postInstall = ''
            # Install desktop file
            mkdir -p $out/share/applications
            cat > $out/share/applications/org.codeberg.lapingvino.Accolade.desktop << EOF
            [Desktop Entry]
            Name=Accolade
            Comment=A distraction-free Fountain editor for screenwriters
            Exec=$out/bin/accolade %F
            Icon=org.codeberg.lapingvino.Accolade
            Terminal=false
            Type=Application
            Categories=Office;WordProcessor;
            MimeType=text/fountain;text/spmd;
            StartupNotify=true
            EOF
            
            # Install MIME type
            mkdir -p $out/share/mime/packages
            cat > $out/share/mime/packages/fountain.xml << EOF
            <?xml version="1.0" encoding="UTF-8"?>
            <mime-info xmlns="http://www.freedesktop.org/standards/shared-mime-info">
              <mime-type type="text/fountain">
                <comment>Fountain screenplay</comment>
                <glob pattern="*.fountain"/>
              </mime-type>
              <mime-type type="text/spmd">
                <comment>Fountain screenplay (legacy)</comment>
                <glob pattern="*.spmd"/>
              </mime-type>
            </mime-info>
            EOF
          '';
          
          meta = with pkgs.lib; {
            description = "A distraction-free Fountain editor for screenwriters";
            homepage = "https://codeberg.org/lapingvino/accolade";
            license = licenses.gpl3Plus;
            maintainers = [ ];
            platforms = platforms.linux;
          };
        };
      });
}