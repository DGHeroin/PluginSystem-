# 设置编译器和标志
GO := go
GOFLAGS := -trimpath

# 输出目录
BUILD_DIR := build
PLUGINS_DIR := $(BUILD_DIR)/plugins

# 目标文件
MASTER := $(BUILD_DIR)/master.exe
PLUGIN1 := $(PLUGINS_DIR)/plugin1.exe
PLUGIN2 := $(PLUGINS_DIR)/plugin2.exe

# 源文件目录
CMD_DIR := cmd

# 默认目标
.PHONY: all
all: $(MASTER) $(PLUGIN1) $(PLUGIN2)

# 创建必要的目录
$(BUILD_DIR) $(PLUGINS_DIR):
	mkdir -p $@

# 编译主程序
$(MASTER): $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $@ $(CMD_DIR)/master/main.go

# 编译插件1
$(PLUGIN1): $(PLUGINS_DIR)
	$(GO) build $(GOFLAGS) -o $@ $(CMD_DIR)/plugins/plugin1/plugin1.go

# 编译插件2
$(PLUGIN2): $(PLUGINS_DIR)
	$(GO) build $(GOFLAGS) -o $@ $(CMD_DIR)/plugins/plugin2/plugin2.go

# 清理构建产物
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# 运行主程序
.PHONY: run-master
run-master: $(MASTER)
	./$(MASTER)

# 运行所有插件
.PHONY: run-plugins
run-plugins: $(PLUGIN1) $(PLUGIN2)
	./$(PLUGIN1) & ./$(PLUGIN2)

# 帮助信息
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Build everything (default)"
	@echo "  clean        - Remove build artifacts"
	@echo "  run-master   - Run the master program"
	@echo "  run-plugins  - Run all plugins"
	@echo "  help         - Show this help message"