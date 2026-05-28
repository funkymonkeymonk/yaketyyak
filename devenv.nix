{ pkgs, inputs, ... }:

let
  projectRoot = builtins.toString ./.;
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
    inputs.llm-agents.packages.${pkgs.system}.pi
  ];

  processes = {
    temporal-dev-server = {
      exec = "temporal server start-dev --ui-port 8233 --http-port 7243";
      process-compose = {
        readiness_probe = {
          exec.command = "temporal operator cluster health --address localhost:7233";
          initial_delay_seconds = 2;
          period_seconds = 2;
          failure_threshold = 10;
        };
      };
    };

    # Single worker — handles both workflow orchestration and activity execution.
    worker = {
      exec = ''
        cd ${projectRoot}/worker && npm ci --silent && npm run build
        exec op run --env-file=${projectRoot}/.env.op -- node ${projectRoot}/worker/dist/worker.js
      '';
      process-compose = {
        availability.restart = "on_failure";
        depends_on."temporal-dev-server".condition = "process_healthy";
        readiness_probe = {
          exec.command = "curl -sf http://localhost:''${HEALTH_PORT:-8080}/health";
          initial_delay_seconds = 5;
          period_seconds = 3;
          failure_threshold = 10;
        };
      };
    };
  };

  containers."worker" = {
    name = "yyx-worker";
    copyToRoot = null;
    startupCommand = "op run --env-file=${projectRoot}/.env.op -- node ${projectRoot}/worker/dist/worker.js";
  };

  scripts.yyx.exec = "${projectRoot}/dist/yyx";

  scripts.dev-run.exec = ''
    set -e
    echo "Building..."
    mkdir -p dist
    go build -o dist/yyx .
    echo "Launching TUI..."
    exec dist/yyx
  '';

  tasks = {
    "yyx:build" = {
      description = "Build yyx TUI binary and TypeScript worker";
      exec = ''
        mkdir -p dist
        go build -o dist/yyx .
        cd worker && npm ci --silent && npm run build
      '';
    };



    "yyx:test" = {
      description = "Run all tests";
      exec = ''
        go test . ./cmd/... ./temporal/... ./tui/...
      '';
    };

    "yyx:lint" = {
      description = "Run go vet";
      exec = ''
        go vet . ./cmd/... ./temporal/... ./tui/...
      '';
    };

    "yyx:check" = {
      description = "Run lint and tests";
      after = [ "yyx:lint" "yyx:test" ];
      exec = ''
        echo "All checks passed."
      '';
    };

    "yyx:install" = {
      description = "Full pipeline: lint, test, build, install yyx TUI";
      after = [ "yyx:check" "yyx:build" ];
      exec = ''
        install -d "$GOPATH/bin"
        install -m 755 dist/yyx "$GOPATH/bin/yyx"
        echo "Installed yyx to $GOPATH/bin"
      '';
    };

    "utils:clean" = {
      description = "Remove build artifacts";
      exec = ''
        rm -rf dist/
        go clean
      '';
    };
  };

  enterShell = ''
    echo "✦ yaketyyak dev environment"
    echo "  Go: $(go version 2>/dev/null || echo 'not found')"
    echo "  yyx: $(which yyx 2>/dev/null || echo 'run \`devenv tasks run yyx:install\` first')"
    echo ""
    echo "  Available tasks:"
    echo "    devenv tasks list                          # discover all tasks"
    echo "    dev-run                                    # build and run TUI (fast)"
    echo "    devenv tasks run yyx:build                 # build only"
    echo "    devenv tasks run yyx:install               # full pipeline (lint+test+build+install)"
    echo "    devenv tasks run yyx:test                  # run tests"
    echo "    devenv tasks run yyx:lint                  # go vet"
    echo "    devenv tasks run utils:clean               # clean artifacts"
    echo ""
    echo "  Dev server:"
    echo "    devenv up                            Start dev server + worker"
    echo "    zellij --layout ${projectRoot}/devenv/zellij/layout.kdl  TUI with shell + devenv up"
    echo ""
    echo "  Workflow:"
    echo "    yyx shave <yak>                      Start a shave workflow"
    echo "    dist/yyx shave <yak>                 (if yyx not installed)"
    echo ""
    echo "  MCP server (for AI assistants):"
    echo "    devenv mcp                            stdio mode"
    echo "    devenv mcp --http 8080                HTTP mode"
  '';
}
