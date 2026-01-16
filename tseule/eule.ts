export default class Eule {
  constructor() {
    console.log("lol");
  }

  main(args: string[]) {
    if (args.length < 2) {
      this.repl();
    } else {
      this.file(args[1]);
    }
  }

  private file(_path: string) {}

  private repl() {}
}
