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
        
        # GTK4 and related libraries
        gtkLibs = with pkgs; [
          gtk4
          libadwaita
          gtksourceview5
          webkitgtk_4_1
          glib
          gobject-introspection
          pkg-config
        ];
        
        # Build tools
        buildInputs = with pkgs; [
          go
          gcc
          pkg-config
          wrapGAppsHook4
        ] ++ gtkLibs;
        
        # Runtime dependencies
        nativeBuildInputs = with pkgs; [
          pkg-config
          wrapGAppsHook4
          gobject-introspection
        ];
        
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = buildInputs;
          nativeBuildInputs = nativeBuildInputs;
          
          shellHook = ''
            export CGO_ENABLED=1
            export PKG_CONFIG_PATH="${pkgs.lib.makeSearchPath "lib/pkgconfig" gtkLibs}"
            export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath gtkLibs}:$LD_LIBRARY_PATH"
            export GI_TYPELIB_PATH="${pkgs.lib.makeSearchPath "lib/girepository-1.0" gtkLibs}"
            export XDG_DATA_DIRS="${pkgs.lib.makeSearchPath "share" gtkLibs}:$XDG_DATA_DIRS"
            
            echo "Accolade development environment activated"
            echo "Go version: $(go version)"
            echo "GTK4 development libraries are available"
            echo ""
            echo "To build: go build -v ."
            echo "To run: ./accolade"
          '';
        };
        
        packages.default = pkgs.buildGoModule {
          pname = "accolade";
          version = "0.1.0";
          src = ./.;
          
          vendorHash = null;
          
          buildInputs = gtkLibs;
          nativeBuildInputs = nativeBuildInputs;
          
          CGO_ENABLED = 1;
          
          preBuild = ''
            export PKG_CONFIG_PATH="${pkgs.lib.makeSearchPath "lib/pkgconfig" gtkLibs}"
          '';
          
          postInstall = ''
            wrapGAppsHook4 $out/bin/accolade
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