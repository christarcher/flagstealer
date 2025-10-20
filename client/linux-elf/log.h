// log.h

#ifndef LOG_H
#define LOG_H

#include <stdio.h>
#include <stdarg.h>

#ifdef DEBUG
    #define LOG_DEBUG(fmt, ...) \
        do { \
            fprintf(stderr, "[DEBUG] %s:%d:%s(): " fmt "\n", \
                __FILE__, __LINE__, __func__, ##__VA_ARGS__); \
        } while (0)

#else
    // 非 DEBUG 构建时，宏为空，不输出、不执行任何逻辑
    #define LOG_DEBUG(fmt, ...) do {} while (0)

#endif // DEBUG

#endif // LOG_H