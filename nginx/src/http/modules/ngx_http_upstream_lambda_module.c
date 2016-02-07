
/*
 * Copyright (C) Igor Sysoev
 * Copyright (C) Nginx, Inc.
 */


#include <ngx_config.h>
#include <ngx_core.h>
#include <ngx_http.h>
#include <assert.h>


typedef struct {
    /* the round robin data must be first */
    ngx_http_upstream_rr_peer_data_t   rrp;

    ngx_uint_t                         hash;

    u_char                             addrlen;
    u_char                            *addr;

    u_char                             tries;
} ngx_http_upstream_lambda_peer_data_t;


static ngx_int_t ngx_http_upstream_init_lambda_peer(ngx_http_request_t *r,
    ngx_http_upstream_srv_conf_t *us);
static ngx_int_t ngx_http_upstream_get_lambda_peer(ngx_peer_connection_t *pc,
    void *data);
static char *ngx_http_upstream_lambda(ngx_conf_t *cf, ngx_command_t *cmd,
    void *conf);


static ngx_command_t  ngx_http_upstream_lambda_commands[] = {

    { ngx_string("lambda"),
      NGX_HTTP_UPS_CONF|NGX_CONF_NOARGS,
      ngx_http_upstream_lambda,
      0,
      0,
      NULL },

      ngx_null_command
};


static ngx_http_module_t  ngx_http_upstream_lambda_module_ctx = {
    NULL,                                  /* preconfiguration */
    NULL,                                  /* postconfiguration */

    NULL,                                  /* create main configuration */
    NULL,                                  /* init main configuration */

    NULL,                                  /* create server configuration */
    NULL,                                  /* merge server configuration */

    NULL,                                  /* create location configuration */
    NULL                                   /* merge location configuration */
};


ngx_module_t  ngx_http_upstream_lambda_module = {
    NGX_MODULE_V1,
    &ngx_http_upstream_lambda_module_ctx, /* module context */
    ngx_http_upstream_lambda_commands,    /* module directives */
    NGX_HTTP_MODULE,                       /* module type */
    NULL,                                  /* init master */
    NULL,                                  /* init module */
    NULL,                                  /* init process */
    NULL,                                  /* init thread */
    NULL,                                  /* exit thread */
    NULL,                                  /* exit process */
    NULL,                                  /* exit master */
    NGX_MODULE_V1_PADDING
};


static u_char ngx_http_upstream_lambda_pseudo_addr[3];


static ngx_int_t
ngx_http_upstream_init_lambda(ngx_conf_t *cf, ngx_http_upstream_srv_conf_t *us)
{
    if (ngx_http_upstream_init_round_robin(cf, us) != NGX_OK) {
        return NGX_ERROR;
    }

    us->peer.init = ngx_http_upstream_init_lambda_peer;

    return NGX_OK;
}

char *pstr(ngx_str_t str) {
    char *s = malloc(str.len+1);
    assert(s);
    memcpy(s, str.data, str.len);
    s[str.len] = '\0';
    return s;
}

ngx_str_t arg_search(ngx_str_t *s, char *key) {
    uint32_t start = 0;
    uint32_t end = 0;
    uint32_t split = 0;
    ngx_str_t value = {.data=NULL, .len=0};
    // iterate over parts separated by '&'
    for (;;) {
        if (end == s->len || s->data[end] == '&') {
            ngx_str_t kv = {.data = &s->data[start], .len = end-start};
            // find '='
            for (split=0; split < kv.len; split++) {
                if (kv.data[split] == '=') {
                    if (strncmp((char *)kv.data, key, split) == 0) {
                        value.data = &kv.data[split+1];
                        value.len = kv.len - (split+1);
                        break;
                    }
                }
            }

            if (end == s->len)
                break;
            start = end+1;
        }
        end++;
    }
    return value;
}

static ngx_int_t
ngx_http_upstream_init_lambda_peer(ngx_http_request_t *r,
    ngx_http_upstream_srv_conf_t *us)
{
    struct sockaddr_in                     *sin;
#if (NGX_HAVE_INET6)
    struct sockaddr_in6                    *sin6;
#endif
    ngx_http_upstream_lambda_peer_data_t  *iphp;
    ngx_str_t db_key = arg_search(&r->args, "key");

    iphp = ngx_palloc(r->pool, sizeof(ngx_http_upstream_lambda_peer_data_t));
    if (iphp == NULL) {
        return NGX_ERROR;
    }

    r->upstream->peer.data = &iphp->rrp;

    if (ngx_http_upstream_init_round_robin_peer(r, us) != NGX_OK) {
        return NGX_ERROR;
    }

    r->upstream->peer.get = ngx_http_upstream_get_lambda_peer;

    switch (r->connection->sockaddr->sa_family) {

    case AF_INET:
        sin = (struct sockaddr_in *) r->connection->sockaddr;
        iphp->addr = (u_char *) &sin->sin_addr.s_addr;
        iphp->addrlen = 3;
        break;

#if (NGX_HAVE_INET6)
    case AF_INET6:
        sin6 = (struct sockaddr_in6 *) r->connection->sockaddr;
        iphp->addr = (u_char *) &sin6->sin6_addr.s6_addr;
        iphp->addrlen = 16;
        break;
#endif

    default:
        iphp->addr = ngx_http_upstream_lambda_pseudo_addr;
        iphp->addrlen = 3;
    }

    iphp->hash = 89;
    iphp->tries = 0;

    return NGX_OK;
}


static ngx_int_t
ngx_http_upstream_get_lambda_peer(ngx_peer_connection_t *pc, void *data)
{
    ngx_http_upstream_lambda_peer_data_t  *iphp = data;

    ngx_uint_t                    p, hash;
    ngx_http_upstream_rr_peer_t  *peer;

    printf("TYLER: tries: %d\n", iphp->tries);
    iphp->tries++;

    hash = iphp->hash;
    p = hash % iphp->rrp.peers->number;
    peer = &iphp->rrp.peers->peer[p];

    // init connection info
    pc->cached = 0;
    pc->connection = NULL;
    pc->sockaddr = peer->sockaddr;
    pc->socklen = peer->socklen;
    pc->name = &peer->name;

    return NGX_OK;
}


static char *
ngx_http_upstream_lambda(ngx_conf_t *cf, ngx_command_t *cmd, void *conf)
{
    ngx_http_upstream_srv_conf_t  *uscf;

    uscf = ngx_http_conf_get_module_srv_conf(cf, ngx_http_upstream_module);

    if (uscf->peer.init_upstream) {
        ngx_conf_log_error(NGX_LOG_WARN, cf, 0,
                           "load balancing method redefined");
    }

    uscf->peer.init_upstream = ngx_http_upstream_init_lambda;

    uscf->flags = NGX_HTTP_UPSTREAM_CREATE
                  |NGX_HTTP_UPSTREAM_WEIGHT
                  |NGX_HTTP_UPSTREAM_MAX_FAILS
                  |NGX_HTTP_UPSTREAM_FAIL_TIMEOUT
                  |NGX_HTTP_UPSTREAM_DOWN;

    return NGX_CONF_OK;
}
