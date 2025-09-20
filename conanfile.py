from conan import ConanFile
from conan.tools.cmake import CMake, CMakeToolchain, CMakeDeps, cmake_layout
from conan.tools.files import copy, get, chdir
import os

class PrepushConan(ConanFile):
    name = "pre-push"
    version = "0.1.0"
    description = "Build tools - pre-push: git describe CLI"
    license = "Apache 2.0"
    author = "AlexBurnes"
    url = "https://github.com/AlexBurnes/pre-push-go"
    topics = ("roam", "clearing", "management", "golang")
    
    # Binary configuration
    settings = "os", "compiler", "build_type", "arch"
    options = {
        "shared": [True, False],
        "fPIC": [True, False],
    }
    default_options = {
        "shared": False,
        "fPIC": True,
    }
    
    # Sources are located in the same place as this recipe, copy them to the recipe
    exports_sources = "CMakeLists.txt", "cmd/*", "pkg/*", "*.go", "go.mod", "go.sum"
    
    def build_requirements(self):
        # Go as build requirement
        self.tool_requires("golang/1.23.0")
        
        # CMake as build requirement
        self.tool_requires("cmake/[>=3.16]")
    
    def config_options(self):
        if self.settings.os == "Windows":
            del self.options.fPIC
    
    def configure(self):
        if self.options.shared:
            self.options.rm_safe("fPIC")
    
    def layout(self):
        cmake_layout(self)
    
    def generate(self):
        tc = CMakeToolchain(self)
        tc.generate()
        
        deps = CMakeDeps(self)
        deps.generate()
    
    def build(self):
        cmake = CMake(self)
        cmake.configure()
        cmake.build()
    
    def package(self):
        cmake = CMake(self)
        cmake.install()
        
        # Copy additional files
        copy(self, "*.yaml", src=os.path.join(self.source_folder, "config"), 
             dst=os.path.join(self.package_folder, "etc"))
    
    def package_info(self):
        self.cpp_info.libs = []
        self.cpp_info.includedirs = []
        
        # Set binary path
        self.cpp_info.bindirs = ["bin"]
        
        # Set config path
        self.cpp_info.resdirs = ["etc"]