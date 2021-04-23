import logging

logging.info(r"Loads 'do_app.py' module")


def app_main(data_queue):
    import app
    app.main(data_queue)
