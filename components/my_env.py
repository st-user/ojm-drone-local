from dotenv import dotenv_values

my_env = dotenv_values('.env')


def get_env(key: str, default_value: str = '') -> str:
    if key in my_env:
        ret = my_env[key]
        return ret if ret is not None else default_value
    return default_value
