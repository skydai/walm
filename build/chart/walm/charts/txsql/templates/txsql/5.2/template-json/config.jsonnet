{
  // system level params
  Transwarp_License_Address: '',
  Transwarp_Install_ID: '',
  Transwarp_Install_Namespace: '',
  Transwarp_Ingress: {
    http_port: 80,
    https_port: 443,
  },
  Customized_Instance_Selector: {},
  Customized_Namespace: '',
  Transwarp_Network_EnableStaticIP: true,

  // Transwarp_Application_Pause,
  App: {
    txsql: {
      priority: 0,
      replicas: 3,
      image: '172.16.1.99/gold/txsql:transwarp-5.2',
      env_list: [],
      use_host_network: false,
      resources: {
        cpu_limit: 1,
        cpu_request: 0.5,
        memory_limit: 3,
        memory_request: 1,
        storage: {
          data: {
            storageClass: 'silver',
            size: '100Gi',
            accessModes: ['ReadWriteOnce'],
            limit: {},
          },
          log: {
            storageClass: 'silver',
            size: '20Gi',
            accessModes: ['ReadWriteOnce'],
            limit: {},
          },
        },
      },
    },
  },

  Transwarp_Config: {
    Transwarp_Auto_Injected_Volumes: [],
    Ingress: {},
  },

  Advance_Config: {
    txsql: {
      // txsqlnodes
    },
    db_properties: {
      'db.driver': 'com.mysql.jdbc.Driver',
      'db.user': 'root',
      'db.password': '123456',
    },
    install_conf: {
      // TxSQLNodes=(172.16.1.51 172.16.1.52 172.16.1.53)
      DATA_DIR: '/var/txsqldata',
      LOG_DIR: '/var/txsqllog',

      SQL_RW_PORT: 3306,
      BINLOG_PORT: 6000,
      MYSQL_LOCAL_PORT: 13306,
      BINLOGSVR_RPC_PORT: 17000,
      PAXOS_PORT: 8001,
    },
  },
}