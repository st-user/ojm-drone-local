import multiprocessing as mp
import logging
import time

logging.basicConfig(level=logging.INFO)

logging.info(r"Loads 'run.py' module")


def do_main():

    """
        Runs the main routine of this application.
        The main routine is called on the forked proccess
        so that it can be restarted
        (the old process is killed and a new one starts)
        automatically.
    """

    mp.set_start_method('spawn')
    import do_app
    while True:
        data_queue = mp.Queue()
        p = mp.Process(target=do_app.app_main, args=(data_queue,))
        try:
            logging.info('Start proc.')
            p.start()
            stopped_pid = data_queue.get()
            logging.info(f'End proc({stopped_pid}).')
            p.kill()
        finally:
            logging.info('Kill proc.')
            time.sleep(3)


if __name__ == '__main__':
    do_main()
