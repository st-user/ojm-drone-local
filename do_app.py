import logging
import multiprocessing as mp

logging.info(r"Loads 'do_app.py' module")


def app_main(data_queue: mp.Queue) -> None:
    import app
    app.main(data_queue)
