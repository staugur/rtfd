from flask_pluginkit import PluginManager, Flask
app = Flask(__name__)
PluginManager(app, plugin_packages=["rtfd"])
