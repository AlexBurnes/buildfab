from conan import ConanFile
from conan.tools.files import get, copy
from conan.tools.layout import basic_layout
from conan.errors import ConanInvalidConfiguration
import os

class GolangConan(ConanFile):
    name = "golang"
    version = "1.23.0"
    description = "Go toolchain (compiler + tools)"
    license = "BSD-3-Clause"
    url = "https://go.dev"
    settings = "os", "arch"
    package_type = "application"   # это тул, а не C/C++ lib
    # этот пакет обычно используют как tool_requires (build-require)

    def layout(self):
        basic_layout(self)

    def _go_os_arch(self):
        # сопоставление Conan -> названия в архивах Go
        os_map = {
            "Windows": "windows",
            "Linux": "linux",
            "Macos": "darwin",
        }
        arch_map = {
            "x86_64": "amd64",
            "armv8": "arm64",
            "armv7": "armv6l",  # при необходимости скорректируй под свои цели
            "armv6": "armv6l",
            "ppc64le": "ppc64le",
        }
        goos = os_map.get(str(self.settings.os))
        goarch = arch_map.get(str(self.settings.arch))
        if not goos or not goarch:
            raise ConanInvalidConfiguration(f"Unsupported os/arch: {self.settings.os}/{self.settings.arch}")
        ext = "zip" if str(self.settings.os) == "Windows" else "tar.gz"
        return goos, goarch, ext

    def build(self):
        goos, goarch, ext = self._go_os_arch()
        filename = f"go{self.version}.{goos}-{goarch}.{ext}"
        url = f"https://go.dev/dl/{filename}"

        # Скачаем и распакуем в build_folder. Архив содержит корневую папку "go"
        # Пропускаем проверку sha256 ради простоты (можно добавить позже через sha256=...).
        get(self, url, strip_root=True)  # положит содержимое "go/" прямо в current folder

    def package(self):
        # Всё распаковано в self.build_folder после strip_root=True:
        # копируем в package_folder
        copy(self, pattern="*", src=self.build_folder, dst=self.package_folder)

    def package_info(self):
        bin_path = os.path.join(self.package_folder, "bin")
        # Для Conan 2: добавляем в PATH для build-окружения и run-окружения.
        self.buildenv_info.append_path("PATH", bin_path)
        self.runenv_info.append_path("PATH", bin_path)