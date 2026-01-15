from lib import *

import os
import time

from datetime import datetime


TEMP_NAME = "__test_build.exe"
BENCH_FOLDER = "../tests"
RESULTS_DIR = "../benchmarks_results"
RESULT_FILE = "result"
FORMAT = "{}\npytime:\n{}\n\nprogram stdout:\n{}\n"
EXTS = ["eulb"]
GO_CWD = "./goeule"
GO_MAIN = "."


if __name__ == "__main__":
    os.chdir(GO_CWD)
    os.system(f"go build -o {TEMP_NAME} {GO_MAIN}")

    try:
        os.makedirs(RESULTS_DIR, exist_ok=True)
        file_name = (
            f"{RESULTS_DIR}/{RESULT_FILE}"
            + "_"
            + f"{datetime.now().strftime('%Y_%m_%d_%H_%M')}"
        )
        with open(file_name + ".txt", "w", encoding="utf-8") as bmr_file:
            for file in read_files(BENCH_FOLDER, EXTS):
                print(file, "-> ...", end="", flush=True)
                start = time.perf_counter()
                stdout, stderr = run_script(TEMP_NAME, file)
                perf = time.perf_counter() - start

                if stderr != "":
                    print(f"\b\b\berror at {file}: {short(stderr, 32)}")
                    continue

                result = FORMAT.format(cover(str(file), 80), perf, stdout)
                bmr_file.write(result)
                print("\b\b\bdone!")

        print("all done!")
    finally:
        os.unlink(TEMP_NAME)
