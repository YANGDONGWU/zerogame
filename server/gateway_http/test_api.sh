#!/bin/bash

# HTTP网关API测试脚本

echo "=== HTTP网关API测试 ==="
echo

# 测试健康检查
echo "1. 健康检查测试"
curl -s http://localhost:8888/health | jq .
echo -e "\n"

# 测试RESTful风格调用 - 用户信息查询 (小写)
echo "2. RESTful风格 - 获取用户信息 (小写)"
curl -s -X GET "http://localhost:8888/api/user/getUserInfo?user_id=123" | jq .
echo -e "\n"

# 测试大小写转换 - 大写服务名
echo "2b. 大小写转换测试 - UserService"
curl -s -X GET "http://localhost:8888/api/UserService/getUserInfo?user_id=123" | jq .
echo -e "\n"

# 测试RESTful风格调用 - 用户登录 (JSON body)
echo "3. RESTful风格 - 用户登录 (JSON body)"
response=$(curl -s -X POST http://localhost:8888/api/login/logon \
  -H "Content-Type: application/json" \
  -d '{"accounts": "testuser", "password": "testpass"}')
echo "$response"
if [[ $response == \{* ]]; then
  echo "$response" | jq .
else
  echo "(Response is not JSON)"
fi
echo -e "\n"

# 测试RESTful风格调用 - 用户登录 (查询参数)
echo "3b. RESTful风格 - 用户登录 (查询参数)"
curl -s "http://localhost:8888/api/login/logon?accounts=testuser&password=testpass" | jq .
echo -e "\n"

# 测试表单数据 (application/x-www-form-urlencoded)
echo "3c. RESTful风格 - 用户登录 (表单数据)"
curl -s -X POST http://localhost:8888/api/login/logon \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "accounts=testuser&password=testpass" | jq .
echo -e "\n"

# 测试混合模式 (JSON body + 查询参数)
echo "3d. RESTful风格 - 用户登录 (混合模式)"
response=$(curl -s "http://localhost:8888/api/login/logon?accounts=testuser" \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"password": "testpass"}')
echo "$response"
if [[ $response == \{* ]]; then
  echo "$response" | jq .
else
  echo "(Response is not JSON)"
fi
echo -e "\n"

# 测试通用网关调用
echo "4. 通用网关 - 获取用户信息"
curl -s -X POST http://localhost:8888/api/generic \
  -H "Content-Type: application/json" \
  -d '{
    "service": "user",
    "method": "getUserInfo",
    "data": {
      "user_id": 456
    }
  }' | jq .
echo -e "\n"

# 测试GET查询参数调用
echo "5. GET查询参数 - 用户信息"
response=$(curl -s "http://localhost:8888/api/generic?service=user&method=getUserInfo&user_id=789")
echo "$response"
if [[ $response == \{* ]]; then
  echo "$response" | jq .
else
  echo "(Response is not JSON)"
fi
echo -e "\n"

# 测试错误情况
echo "6. 错误测试 - 服务不存在"
curl -s -X POST http://localhost:8888/api/generic \
  -H "Content-Type: application/json" \
  -d '{
    "service": "nonexist",
    "method": "test"
  }' | jq .
echo -e "\n"

echo "=== 测试完成 ==="
