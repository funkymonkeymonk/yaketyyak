{ pkgs, ... }:

let
  projectRoot = builtins.toString ./.;

  tempo = pkgs.buildGoModule rec {
    pname = "tempo";
    version = "0.1.14";
    src = pkgs.fetchFromGitHub {
      owner = "galaxy-io";
      repo = "tempo";
      rev = "v${version}";
      hash = "sha256-HjKBm34HGX11mQCmdMpa09jT6ZgqpRO6gJaFmCWryLQ=";
    };
    vendorHash = "sha256-P7T9yQD28/MTTYnvhb/uckyLh15I7AD7uMnmaTa3YZk=";
    subPackages = [ "cmd/tempo" ];
    ldflags = [ "-s" "-w" ];
  };
in
{
  languages.go = {
    enable = true;
    enableHardeningWorkaround = true;
  };

  packages = with pkgs; [
    gopls
    go-tools
    temporal-cli
    gotools
    zellij
    tempo
    tree-sitter-grammars.tree-sitter-kdl
    kdlfmt
    nodejs
  ];

  processes = {
    temporal-dev-server.exec = "temporal server start-dev --db-filename ${projectRoot}/.temporal.db";

    temporal-setup = {
      exec = ''
        # Wait for Temporal to be ready, then register custom search attributes.
        until temporal operator namespace describe default >/dev/null 2>&1; do
          sleep 1
        done
        temporal operator search-attribute create \
          --name YakName --type Text \
          --name Phase   --type Keyword \
          --name PrUrl   --type Keyword \
          --namespace default \
          2>/dev/null || true
        echo "Temporal search attributes ready"
      '';
      process-compose = {
        availability.restart = "no";
        depends_on."temporal-dev-server".condition = "process_started";
      };
    };

    worker = {
      exec = "npm --prefix ${projectRoot}/worker install --silent && npm --prefix ${projectRoot}/worker run build && npm --prefix ${projectRoot}/worker run start";
      process-compose = {
        availability.restart = "on_failure";
        availability.max_restarts = 3;
        replicas = 1;
        depends_on."temporal-setup".condition = "process_completed_successfully";
      };
    };
  };

  scripts.yy.exec = "${projectRoot}/dist/yy";

  scripts.dev-run.exec = ''
    set -e
    echo "Building..."
    mkdir -p dist
    go build -o dist/yy .
    echo "Launching TUI..."
    exec ./dist/yy
  '';

  scripts.dev-session.exec = ''
    exec zellij --layout ${projectRoot}/devenv/zellij/layout.kdl
  '';

  tasks = {
    "yy:build" = {
      description = "Build the yy binary (fast, no lint/test)";
      exec = ''
        mkdir -p dist
        go build -o dist/yy .
      '';
    };

    "yy:test" = {
      description = "Run all tests";
      exec = ''
        go test ./cmd/... ./temporal/...
      '';
    };

    "yy:lint" = {
      description = "Run go vet";
      exec = ''
        go vet ./cmd/... ./temporal/...
      '';
    };

    "yy:check" = {
      description = "Run lint and tests";
      after = [ "yy:lint" "yy:test" ];
      exec = ''
        echo "All checks passed."
      '';
    };

    "yy:install" = {
      description = "Full pipeline: lint, test, build, install";
      after = [ "yy:check" "yy:build" ];
      exec = ''
        install -d "$GOPATH/bin"
        install -m 755 dist/yy "$GOPATH/bin/yy"
        echo "Installed yy to $GOPATH/bin/yy"
      '';
    };

    "utils:clean" = {
      description = "Remove build artifacts";
      exec = ''
        rm -rf dist
        go clean
      '';
    };
  };

  enterShell = ''
    echo "✦ yaketyyak dev environment"
    echo "  Go: $(go version 2>/dev/null || echo 'not found')"
    echo "  yy: $(which yy 2>/dev/null || echo 'run \`devenv tasks run project:setup\` first')"
    echo ""
    echo "  Available tasks:"
    echo "    devenv tasks list                          # discover all tasks"
    echo "    dev-run                                    # build and run TUI (fast)"
    echo "    devenv tasks run yy:build                 # build only"
    echo "    devenv tasks run yy:install               # full pipeline (lint+test+build+install)"
    echo "    devenv tasks run yy:test                  # run tests"
    echo "    devenv tasks run yy:lint                  # go vet"
    echo "    devenv tasks run utils:clean               # clean artifacts"
    echo ""
    echo "  Dev server:"
    echo "    devenv up                            Start dev server + worker"
    echo "    dev-session                          Launch zellij TUI (gh-dash + dev server + opencode + tempo)"
    echo "    zellij --layout ${projectRoot}/devenv/zellij/layout.kdl  Direct zellij layout"
    echo ""
    echo "  Workflow:"
    echo "    yy start --repo user/repo --repo-root /path          Start a workflow"
    echo ""
    echo "  MCP server (for AI assistants):"
    echo "    devenv mcp                            stdio mode"
    echo "    devenv mcp --http 8080                HTTP mode"
  '';
}
