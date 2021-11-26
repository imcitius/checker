job "{$ UniqName .I $}" {
  datacenters = [
    "{$ index .I.Datacenters 0 $}"
  ]

  type = "service"

  meta {
    unixtime            = "{$ unixtime $}"
    link_nomad          = "http://nomad.service.{$ index .I.Datacenters 0 $}.consul:4646/ui/jobs/{$ UniqName .I $}"
    link_project        = "{$ .P.gitlab_project $}"
    link_commit         = "{$ .P.gitlab_project $}/commit/{$ .P.commit_sha $}"
    text_deploy_by      = "{$ .P.deploy_by $}"
    text_commit_sha     = "{$ .P.commit_sha $}"
    text_commit_message = <<EOH
{$ .P.commit_message $}
EOH
  }

  /*update {
    max_parallel = 1
    min_healthy_time = "15s"
    healthy_deadline = "2m"
    progress_deadline = "5m"
  }*/

  reschedule {
    attempts       = 5
    interval       = "5m"
    delay          = "5s"
    delay_function = "exponential"
    max_delay      = "30s"
    unlimited      = false
  }

  migrate {
    max_parallel     = 1
    min_healthy_time = "15s"
    healthy_deadline = "5m"
  }

  vault {
    policies = [
      "read"
    ]
  }

  group "checkers" {
    count = 1

    restart {
      attempts = 2
      delay    = "5s"
    }

    task "checker" {
      driver = "docker"

      meta {
#        version = "{$ .P.version $}"
      }

      template {
        data        = <<EOH
CHECKER_CONFIG = "{{ key .P.consul_path }}"
CONSUL_ADDR = "http://consul.service.{$ index .I.Datacenters 0 $}.consul:8500"
VAULT_ADDR = "https://vault.service.infra1.consul"
EOH
        env         = true
        destination = "secrets/.env"
      }

      config {
        force_pull   = true
        image        = "{$ .P.version $}"
        network_mode = "weave"
        command      = "/app/checker"

        args = [
          "check",
          "-s",
          "env"
        ]

        port_map = {
          http = "80"
        }

        logging {
          type = "fluentd"
          config {
            fluentd-address = "localhost:24226"
            tag             = "checker"
          }
        }
      }

      service {
        address_mode = "driver"
        name         = "checker"

        check {
          address_mode   = "driver"
          port           = "80"
          type           = "http"
          path           = "/healthcheck"
          method         = "GET"
          interval       = "5s"
          timeout        = "1s"
          initial_status = "passing"
        }

        check_restart {
          limit           = 5
          grace           = "15s"
          ignore_warnings = true
        }
      }

      resources {
        cpu    = 50
        memory = 64

        network {
          mbits = 1
        }
      }
    }
  }
}
