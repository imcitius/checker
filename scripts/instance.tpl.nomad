job "{$ UniqName .I $}" {
  datacenters = [
    "{$ index .I.Datacenters 0 $}"
  ]

  type = "service"

  meta {
    unixtime            = "{$ unixtime $}"
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
        version = "{$ .P.version $}"
      }

      template {
        data        = <<EOH
CONSUL_ADDR = "http://consul.service.{$ index .I.Datacenters 0 $}.consul:8500"
VAULT_ADDR = "https://vault.service.infra1.consul"
EOH
        env         = true
        destination = "secrets/.env"
      }

      template {
        data        = <<EOH
{{ key "{$ .P.consul_path $}" }}
EOH
        env         = false
        destination = "secrets/config"
      }

      config {
        force_pull   = true
        image        = "{$ .P.image $}"
        network_mode = "weave"
        command      = "/app/checker"

        args = [
          "check",
          "-f",
          "/secrets/config"
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
